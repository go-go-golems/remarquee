package mdpdf

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultHeaderLoadsMathSymbols(t *testing.T) {
	if !strings.Contains(defaultLatexHeader, `\usepackage{stmaryrd}`) {
		t.Fatal("default header must provide llbracket/rrbracket")
	}
}

func TestMathPDFDirectAndBundle(t *testing.T) {
	for _, tool := range []string{"pandoc", "xelatex", "kpsewhich"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("optional PDF integration dependency absent: %s", tool)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	packagePath, err := exec.CommandContext(ctx, "kpsewhich", "stmaryrd.sty").Output()
	if err != nil || strings.TrimSpace(string(packagePath)) == "" {
		t.Skip("optional PDF integration dependency absent: stmaryrd")
	}

	const math = "$$\n\\operatorname{SendResult} = \\operatorname{Sent}\n+ \\operatorname{Full}\n$$\n"
	const source = "# Math regression\n\n" + math + "\nSemantic brackets: $\\llbracket p \\rrbracket$.\n\n```text\nnot a list\n        - preserve this indentation\n```\n\nOrdinary list\n- item\n"
	for _, bundled := range []bool{false, true} {
		name := "direct"
		if bundled {
			name = "bundle"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input.md")
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if bundled {
				body, err := BuildBundleMarkdown(ctx, []BundleInput{{Path: input, Title: "Math"}}, dir, nil, false)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(body, math) {
					t.Fatalf("bundle changed display math: %q", body)
				}
				input = filepath.Join(dir, "bundle.md")
				if err := os.WriteFile(input, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			output := filepath.Join(dir, "out.pdf")
			if err := ConvertMarkdownFileToPDF(ctx, input, output, DefaultPandocOptions()); err != nil {
				t.Fatal(err)
			}
			pdf, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(pdf), "%PDF-") {
				t.Fatal("conversion did not produce a PDF")
			}
		})
	}
}
