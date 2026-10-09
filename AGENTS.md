# Working on envbuckets

## Workflow

1. Read [PRODUCT.md](docs-eb/PRODUCT.md) and the relevant
   [ticket](docs-eb/tickets/README.md). Continue from its first unfinished phase.
2. Add or review the ticket's phase checklist. Update it after each phase.
3. Research with current Go docs, grep MCP, or DeepWiki when needed. Use APIs
   compatible with `go.mod`.
4. Implement and add unit/integration tests. Follow the user's instructions
   about local test runs versus CI.
5. For Go changes, run `task fix`, review the diff, then `task lint:go`
   (includes `modernize`). Run `task check` when local testing is authorized.
6. Inspect CI with `gh`. Fix failures, review all exit criteria, and record
   evidence or deferred work in the ticket. Update the ticket and index status.
7. Commit and push only when explicitly requested. Include only task changes.

## Safety and quality

- Never read, parse, log, or print real environment-file contents. Tests may
  use synthetic values.
- Keep filesystem operations inside the repository; reject unsafe symlinks.
- Checkout hooks must exit 0 and leave unsafe or missing links alone.
- Fix formatting/lint findings. Suppress only with a nearby explanation.
- Prefer clear standard-library APIs, including `errors.New` and `errors.AsType`
  where supported by `go.mod`.
- Preserve CI and security gates. Keep the golangci-lint pins in `Taskfile.yml`
  and `.github/workflows/ci.yml` aligned. Automatic fixes stay local.
- See [DEVELOPMENT.md](docs-eb/DEVELOPMENT.md) for check details.

## Windows verification

If symlink privilege alone blocks integration tests, local lint/schema/unit
checks plus passing cross-platform CI for the same code are accepted. Record
the limitation and CI link in the ticket. Actual check failures still block.
Symlink setup is deferred; revisit only when requested or the environment changes.

## Commits and releases

- Commit/PR titles: `<gitmoji> <type>(<scope>): <summary>`, for example
  `✨ feat(cli): add readiness checks`.
- `.github/workflows/release.yml` publishes releases; `task snapshot` builds
  artifacts locally without publishing.
