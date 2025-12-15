# Tasks

## TODO

- [x] Add tasks here

- [ ] Implement upload bundle: collate multiple markdown inputs into one PDF with ToC (pandoc wrapper approach)
- [ ] Extend upload md with --preserve-dirs to mirror local directory structure remotely
- [ ] Add upload src: render source files as syntax-highlighted PDFs (code -> markdown -> pandoc)
- [ ] Extend pkg/mdpdf pandoc runner with ToC/highlight options (toc depth, highlight style, listings)
- [ ] Add embedded help docs for bundle and src (pkg/doc/upload/03,04)
- [ ] Add tests: bundle wrapper generation, preserve-dirs path mapping, extension->language mapping
- [ ] Add smoke test scripts: bundle + preserve-dirs + src (upload + cloud ls verification)
