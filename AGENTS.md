# Working on envbuckets

envbuckets is being rebuilt. Read [docs-eb/PRODUCT.md](docs-eb/PRODUCT.md) and
the relevant implementation ticket in [docs-eb/tickets/](docs-eb/tickets/README.md).

- After editing Go files, run `task fix`, review the diff, then run
  `task lint:go`. It includes `modernize`; apply suggestions compatible with
  `go.mod`.
- Use the golangci-lint version pinned in `Taskfile.yml`. Keep that pin and
  `.github/workflows/ci.yml` in sync when upgrading.
- Keep automatic fixes local. CI runs `task lint` without rewriting the
  checkout. See [Developer checks](docs-eb/DEVELOPMENT.md) for the workflow.
- Commit and push only when explicitly requested by the user. Include only
  changes belonging to the task.
- Run `task check` before submitting changes when the environment supports
  symlinks. It runs formatting/lint, schema validation, unit tests, and real-Git
  integration scripts. If Windows fails solely because symlink privilege is
  unavailable, run the available local lint/schema/unit checks and use passing
  cross-platform CI integration results for the same code as the accepted
  alternative. Record the limitation and CI run in the ticket; it does not block
  ticket completion. Actual code, lint, schema, or test failures still block.
- Local environment recorded on 2026-10-09: Windows cannot create symlinks;
  WSL has only `docker-desktop`, and Docker's Linux engine was not running.
  The user deferred Developer Mode, Linux/WSL, and devcontainer setup. Use the
  CI alternative above instead of repeatedly discussing or requesting that
  setup in every session. Revisit setup when the user asks or the environment
  changes.
- Keep the product value-blind: never add code that reads, parses, logs, or
  prints real environment-file contents. Tests may use synthetic values.
- Keep filesystem operations inside the repository and refuse unsafe symlink
  paths.
- Keep the checkout hook nonblocking. It exits 0 and leaves unsafe or missing
  links alone.
- Preserve main-branch CI and security gates.
- Use `<gitmoji> <type>(<scope>): <summary>` for commits and PR titles, such as
  `✨ feat(cli): add readiness checks`. Keep the type and scope meaningful for
  the final change.
- Release publishing is handled by `.github/workflows/release.yml`. `task
  snapshot` builds release artifacts locally without publishing.
