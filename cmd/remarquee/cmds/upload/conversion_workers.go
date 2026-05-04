package upload

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-go-golems/remarquee/pkg/mdpdf"
	"github.com/pkg/errors"
)

type markdownConversionJob struct {
	Input   markdownInput
	PDFName string
	OutPDF  string
}

func buildMarkdownConversionJobs(inputs []markdownInput, outputRoot string, overrideName string, preserveDirs bool) ([]markdownConversionJob, error) {
	jobs := make([]markdownConversionJob, 0, len(inputs))
	for _, in := range inputs {
		pdfName, err := markdownPDFName(in, overrideName, len(inputs))
		if err != nil {
			return nil, err
		}

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
		for _, job := range jobs {
			if err := mdpdf.ConvertMarkdownFileToPDF(ctx, job.Input.AbsPath, job.OutPDF, opts); err != nil {
				return err
			}
		}
		return nil
	}
	if workers > len(jobs) {
		workers = len(jobs)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobCh := make(chan markdownConversionJob)
	errCh := make(chan error, 1)
	var once sync.Once
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for job := range jobCh {
			if err := mdpdf.ConvertMarkdownFileToPDF(ctx, job.Input.AbsPath, job.OutPDF, opts); err != nil {
				once.Do(func() {
					errCh <- err
					cancel()
				})
				return
			}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

sendLoop:
	for _, job := range jobs {
		select {
		case <-ctx.Done():
			break sendLoop
		case jobCh <- job:
		}
	}
	close(jobCh)
	wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		if err := ctx.Err(); err != nil && err != context.Canceled {
			return err
		}
		return nil
	}
}
