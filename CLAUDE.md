# Claude Code

Follow the shared repository instructions in [AGENTS.md](AGENTS.md). For
product behavior, read [docs-eb/PRODUCT.md](docs-eb/PRODUCT.md) and the relevant
ticket in [docs-eb/tickets/](docs-eb/tickets/README.md).

The CLI manages relative links to private files under `.env.d/` using committed
`.envbuckets.json` rules. Commands are `init`, `add`, `switch`, `status`, `branches`,
and `uninstall`; the checkout hook runs automatically. Managed file contents
must never be opened or printed. Use synthetic fixtures for tests.

Run `task fix`, review its diff, then `task lint:go` for Go changes. Full checks
are `task check` and `task security`; npm snapshots are tested with
`task playground`. Windows symlink limits and the accepted CI alternative are
in AGENTS.md. Golden updates require `task test:integration:update` and diff
review. Commit, push, and publish only when the user explicitly requests them.
