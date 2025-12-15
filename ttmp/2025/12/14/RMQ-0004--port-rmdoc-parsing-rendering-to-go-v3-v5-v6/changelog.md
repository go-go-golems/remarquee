# Changelog

## 2025-12-14

- Initial workspace created
- Created initial design doc + diary for the Go port and linked to RMQ-0001 research (deep dive + diary + gap analysis).


## 2025-12-14

Seeded RMQ-0004 from RMQ-0001 research: created design doc for Go rmdoc data model/APIs (with multiple proposals) and started a diary + task list for the port.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/design-doc/01-design-go-rmdoc-data-model-and-apis.md — New design doc for rmdoc data structures + APIs
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/reference/01-diary.md — Started diary for porting work
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/tasks.md — Initial task breakdown


## 2025-12-14

Diary Step 1: recorded seed commit hash (0850623) in RMQ-0004 diary (commit 329318c).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/reference/01-diary.md — Updated Step 1 commit hash


## 2025-12-14

Step 2: implemented pkg/rmdoc (open zip, detect legacy vs cPages, build deterministic page plan + tests). Commit 49acbde.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/content.go — Schema detection + page plan
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/open.go — Archive opening
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/reference/01-diary.md — Added Step 2 with commit hash


## 2025-12-14

Updated tasks: checked off completed Step 2 work (pkg/rmdoc schema detection + page plan + tests) and expanded remaining items into smaller sub-tasks.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/reference/01-diary.md — Step 2 describes the implementation that was checked off
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/tasks.md — Checked off completed items and broke down remaining tasks


## 2025-12-14

Updated design doc to reference current implementation status (pkg/rmdoc) and linked new Go files.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/design-doc/01-design-go-rmdoc-data-model-and-apis.md — Now references pkg/rmdoc implementation (commit 49acbde)


## 2025-12-14

Step 3: added remarquee rmdoc inspect CLI for schema/page-plan debugging (commit c804b36). Verified against repo fixture; cloud smoke test failed due to expired rmapi token.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/rmdoc/inspect.go — New inspect subcommand
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/open.go — Used by inspect
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/reference/01-diary.md — Added Step 3
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/tasks.md — Checked off inspect task


## 2025-12-14

Step 4: added integration tests for pkg/rmdoc.OpenFile using legacy + cPages fixtures (commit 3036c7e).

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/pkg/rmdoc/open_integration_test.go — New integration tests
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/reference/01-diary.md — Added Step 4


## 2025-12-14

Step 5: added remarquee rmdoc render-legacy (rmapi-backed) to render legacy archives to annotated PDFs. Commit 05c257d.

### Related Files

- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/cmd/remarquee/cmds/rmdoc/render_legacy.go — New command
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/reference/01-diary.md — Added Step 5
- /home/manuel/workspaces/2025-12-14/build-remarquee-tool/remarquee/ttmp/2025/12/14/RMQ-0004--port-rmdoc-parsing-rendering-to-go-v3-v5-v6/tasks.md — Checked off legacy render CLI prototype

