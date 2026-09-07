---
Title: Implementation diary
Ticket: RM-MATH-PREPROCESS
Status: active
Topics:
    - markdown
    - pdf
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://AGENT.md
      Note: Repository build and commit instructions
    - Path: repo://lefthook.yml
      Note: Required lint and full-test hooks
    - Path: repo://pkg/mdpdf/literal_regions.go
      Note: Implemented fix at e9d9e77
    - Path: repo://pkg/mdpdf/math_pdf_test.go
      Note: Native PDF proof of regression repair
    - Path: repo://ttmp/2026/09/06/RM-MATH-PREPROCESS--preserve-math-and-code-during-pdf-preprocessing/design-doc/01-math-safe-preprocessing-and-default-symbol-support.md
      Note: Design and validation contract
ExternalSources: []
Summary: Isolated worktree, math/code protection, default symbols, validation and PR delivery.
LastUpdated: 2026-09-07T01:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Diary

## Goal

Implement the two rendering fixes in an isolated remarquee worktree, preserve ordinary list behavior, commit and open a PR.

## Step 1: Add default symbols and shared literal-region protection

Created `/home/manuel/worktrees/remarquee-math-safe-markdown` on `fix/math-safe-markdown` from origin/main at `1ee45a0`. The original checkout was clean and is not edited. Wrote the quick design before implementation, then added stmaryrd to the default header and a shared line-region scanner for display math and fenced code.

Both list-rewriting passes use the scanner, so protection applies during bundle preparation and subsequent PDF conversion. The scanner preserves delimiters/content, handles escaped math closers and fence character/length, and resumes ordinary list formatting after closure. The implementation explicitly does not claim a complete Markdown parser.

### Prompt Context

**User prompt (verbatim):**

```text
ok, create a worktree for remarquee, create a new ticket with a quick design doc, then implement both fixes (add the package per default, fix the parsing). Then commit and open a PR.

[REMINDER] Output a <summary>...</summary> block at the VERY END of your response. This is mandatory.
```

**Assistant interpretation:** Implement the analyzed fixes in a separate worktree, with ticket documentation, tests, commits and a GitHub PR.

**Inferred user intent:** Fix the shared renderer so mathematical documents do not require manual notation/list workarounds.

**Commit (code):** `e9d9e779458849ead417e685e807df93ce24b6b7` — "🐛 Preserve display math and fenced code in PDF preprocessing".

### What I did

- Read repository AGENT.md and existing header, list preprocessing, bundle and test code.
- Added `pkg/mdpdf/literal_regions.go`; updated both list transformations to skip protected lines.
- Loaded `stmaryrd` by default and documented the external dependency and unchanged custom-header replacement behavior.
- Added 18 literal-region cases plus unclosed regions and inline-backtick rejection, checking both preservation and resumed/idempotent list processing.
- Added an unconditional header assertion and real Pandoc/XeLaTeX direct/bundle regressions using multiline `+` equations and double-bracket notation.
- Ran focused tests and both complete affected packages; all passed before attempting the commit.

### Why

Both bundle and direct conversion preprocess lists; fixing only one call site would leave another path broken. Sharing the scanner also prevents flattening from changing indentation inside literal examples. The package is deliberately a default dependency rather than a document-specific workaround.

### What worked

`go test ./pkg/mdpdf -run 'Test(ListPasses|InlineCode|NormalizeListSpacing|FlattenDeepLists|DefaultHeader|MathPDF)' -count=1 -v` passed, including actual direct/bundle PDF generation. `go test ./pkg/mdpdf ./cmd/remarquee/cmds/upload -count=1` passed. `gofmt` and `git diff --check` passed. After generating the ignored frontend assets, the full pre-commit and pre-push hooks passed: `golangci-lint run -v` reported zero issues and `go test ./...` passed. The local frontend build used pnpm 10.15.1 with the frozen lockfile; no lockfile or frontend source changed.

### What didn't work

The first commit was blocked by repository hooks: both lint and full tests reported `cmd/remarquee-ui/embed.go:8:12: pattern frontend/dist: no matching files found`. This is a fresh-worktree build prerequisite, unrelated to math parsing; no commit was created and no hook was bypassed. The existing CI generates frontend assets before lint. A discovery command also queried nonexistent `cmd/remarquee-ui/README.md`; corrected discovery to the actual frontend package and generation entrypoint. Resolved the missing assets with `pnpm --dir cmd/remarquee-ui/frontend install --frozen-lockfile` and `pnpm --dir cmd/remarquee-ui/frontend run build`; verified `dist/index.html` is ignored and retried the commit successfully. Pnpm warned that esbuild install scripts were ignored; the normal build nevertheless completed successfully, without approving additional scripts.

Docmgr doctor warned `unknown_topics — unknown topics value(s): tooling (3 docs)`. Replaced the generic topic with the existing `markdown` and `pdf` vocabulary instead of adding another synonym. The staged whitespace check then flagged a docmgr-generated `new blank line at EOF` in changelog.md; removed that trailing blank line before the docs commit. Final doctor passes.

### What I learned

Fresh worktrees do not inherit ignored embedded frontend assets. The package-related fix changes the default preamble only: a custom header still replaces it and remains responsible for loading its own packages.

### What was tricky to build

Fences must not close on shorter runs, a different character or a delimiter followed by text. Math and code modes must be mutually exclusive. A closing math line can itself resemble a list item, so normalization must also consult the previous line's protected state before deciding whether a real subsequent list needs spacing.

### What warrants a second pair of eyes

Scanner boundaries and the stated scope: line-start dollar/bracket display math and fenced code, not arbitrary raw LaTeX environments or full indented/container syntax. Review dependency behavior on installations without stmaryrd; pure tests still assert the header while optional native PDF integration skips missing external prerequisites.

### What should be done in the future

Review and merge PR #26. Broader Markdown grammar support is outside this focused fix; no implementation task remains for the agreed changes.

### Code review instructions

Start with `literal_regions.go`, then `preprocess.go`, literal-region tests and the real PDF test. Review the default header and README dependency note. No cloud account, upload or physical device is involved in validation.

### Technical details

No public Go API or CLI flag was added. Existing `--latex-header-file` replacement semantics are unchanged. `format_file` is not exposed in this session; Go files are formatted with `gofmt` instead. Generated frontend assets remain ignored and uncommitted. PR: https://github.com/go-go-golems/remarquee/pull/26, branch `fix/math-safe-markdown`. The PR was opened as a draft while finishing ticket documentation; it will be marked ready after that documentation is pushed. The installed CLI and previously uploaded PDFs were not changed.
