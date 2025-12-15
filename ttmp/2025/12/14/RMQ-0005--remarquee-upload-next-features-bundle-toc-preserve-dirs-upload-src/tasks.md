# Tasks

## TODO

- [x] Add tasks here

- [x] Implement upload bundle: collate multiple markdown inputs into one PDF with ToC (pandoc wrapper approach)
- [x] Extend upload md with --preserve-dirs to mirror local directory structure remotely
- [x] Add upload src: render source files as syntax-highlighted PDFs (code -> markdown -> pandoc)
- [x] Extend pkg/mdpdf pandoc runner with ToC options (toc, toc-depth)
- [x] Extend pkg/mdpdf pandoc runner with highlight options (highlight style, listings)
- [ ] Add embedded help docs for bundle and src (pkg/doc/upload/03,04)
- [x] Add tests: bundle wrapper generation + bundle ordering
- [x] Add tests: extension->language mapping
- [ ] Add tests: preserve-dirs path mapping
- [ ] Add smoke test scripts: bundle + preserve-dirs + src (upload + cloud ls verification)
