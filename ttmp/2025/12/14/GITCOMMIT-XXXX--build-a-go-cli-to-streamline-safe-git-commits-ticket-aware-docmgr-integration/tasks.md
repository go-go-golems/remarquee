# Tasks

## TODO

- [ ] Confirm scope: standalone tool vs integrated into existing repo CLIs
- [ ] Decide on configuration approach:
  - [ ] flags-only
  - [ ] optional config file (ticket defaults, path allow/deny rules)
- [ ] Implement `repo` command (git root, branch, clean/dirty summary)
- [ ] Implement `plan` command (preview-only; no writes)
- [ ] Implement `stage` command (explicit paths; optional `--ticket` preset)
- [ ] Implement `commit` command with guardrails:
  - [ ] refuse unrelated staged paths unless `--allow-unrelated`
  - [ ] separate “code” vs “docs” commit modes
  - [ ] print commit hash deterministically
- [ ] Implement docmgr integration (opt-in):
  - [ ] update diary/changelog with commit hash
  - [ ] relate changed files automatically with `--file-note` generation
- [ ] Add tests for path classification + repo discovery
- [ ] Write usage docs + playbook

