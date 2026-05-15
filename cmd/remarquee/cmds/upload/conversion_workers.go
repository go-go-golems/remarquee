package upload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-go-golems/remarquee/pkg/mdpdf"
	"github.com/pkg/errors"
)

type markdownConversionJob struct {
	Input   markdownInput
	PDFName string
	OutPDF  string
}

type conversionError struct {
	job markdownConversionJob
	err error
}

func buildMarkdownConversionJobs(inputs []markdownInput, outputRoot string, overrideName string, preserveDirs bool) ([]markdownConversionJob, error) {
	jobs := make([]markdownConversionJob, 0, len(inputs))
	for _, in := range inputs {
		pdfName, err := markdownPDFName(in, overrideName, len(inputs))
		if err != nil {
			return nil, err
		}
		pdfName = sanitizePDFName(pdfName)

		outPDF := filepath.Join(outputRoot, pdfName)
		if preserveDirs {
			outPDF = filepath.Join(outputRoot, in.RelDir(), pdfName)
			if err := os.MkdirAll(filepath.Dir(outPDF), 0o755); err != nil {
				return nil, errors.Wrap(err, "failed to create output directory structure")
			}
		}

		jobs = append(jobs, markdownConversionJob{
			Input:   in,
			PDFName: pdfName,
			OutPDF:  outPDF,
		})
	}
	return jobs, nil
}

func convertMarkdownJobs(ctx context.Context, jobs []markdownConversionJob, workers int, opts mdpdf.PandocOptions) error {
	if workers < 1 {
		return errors.New("--workers must be at least 1")
	}

	if workers == 1 || len(jobs) <= 1 {
		var convErrs []conversionError
		for _, job := range jobs {
			if err := mdpdf.ConvertMarkdownFileToPDF(ctx, job.Input.AbsPath, job.OutPDF, opts); err != nil {
				convErrs = append(convErrs, conversionError{job: job, err: err})
			}
		}
		if len(convErrs) > 0 {
			return formatConversionErrors(convErrs)
		}
		return nil
	}
	if workers > len(jobs) {
		workers = len(jobs)
	}

	jobCh := make(chan markdownConversionJob)
	var mu sync.Mutex
	var convErrs []conversionError
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for job := range jobCh {
			if err := mdpdf.ConvertMarkdownFileToPDF(ctx, job.Input.AbsPath, job.OutPDF, opts); err != nil {
				mu.Lock()
				convErrs = append(convErrs, conversionError{job: job, err: err})
				mu.Unlock()
			}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)
	wg.Wait()

	if len(convErrs) > 0 {
		return formatConversionErrors(convErrs)
	}
	return nil
}

func formatConversionErrors(errs []conversionError) error {
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, fmt.Sprintf("%s: %v", e.job.Input.AbsPath, e.err))
	}
	return errors.Errorf("%d file(s) failed pandoc conversion: %s", len(errs), strings.Join(msgs, "; "))
}
