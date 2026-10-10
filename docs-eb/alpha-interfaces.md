# T00 shared interfaces and handoff

**Status:** historical alpha handoff, superseded by
[PRODUCT.md](PRODUCT.md) and [EB-01 through EB-10](tickets/README.md). The scope
APIs, branch pins, commands, and exit-code contract below describe the removed
alpha implementation. They are retained as history and must not guide changes
to the current CLI.

Baseline source HEAD: `b670c5048b89a69f3b3b20a291d763ca5b9dc2ae`.
Reviewed baseline paths added in Agent A's worktree: `IMPLEMENTATION_PROMPT.md`
and `IMPLEMENTATION_TICKETS.md`. All other tracked source and tests came from
that HEAD. `CLAUDE.md` and `docs/SPEC.md` were copied individually as ignored
local context, with matching SHA-256 hashes in both worktrees; they are not in
the baseline commit. Baseline commit: `37e9f390073eebd0e7e573d575b458499e2afdf8`.

Baseline checks in Agent A's worktree: `go test ./...`,
`golangci-lint fmt --diff ./...`, `golangci-lint run ./...`, and
`go test -tags integration -run TestScript .` all passed. Some symlink-dependent
tests may skip on Windows; final verification must inspect this explicitly and
run them in a symlink-capable environment.

## Shared code contract

- `(*project).target(branch string) (target, bool, error)` in
  `internal/cli/cmd_hook.go` resolves a supplied branch name without checkout
  or a branch-existence check. It reads a local pin first, then the first
  matching shared rule. `bool` is false when no mapping exists. A malformed pin
  returns an error and does not fall through to a shared rule. `target.bucket`
  is the selected bucket, `target.linked` identifies a local pin,
  `target.via` is `link` or the matching pattern, and `target.priority` is the
  one-based shared-rule position (zero for a pin). Agent B may call this for
  `map explain`; Agent A owns edits to the resolver.
- `(*project).selectScopes(env Env, name string, all, defaultAll bool)
  ([]scope, error)` in `internal/cli/project.go` rejects `--scope` with
  `--all`. `all` selects every configured scope; `defaultAll` selects every
  scope only when no explicit name is supplied. Otherwise it uses the existing
  nearest-scope or explicit-name resolution. Agent A's `check`, `apply`, and
  `status` use all by default; `use` and Agent B's `bucket` commands use one by
  default. `map explain` always describes every configured scope.
- `scope.exists`, `scope.bucketFileExists`, `scope.buckets`, and
  `scope.linkState` in `internal/cli/project.go` provide structural facts
  without reading env values. `linkState` distinguishes missing, real,
  foreign, and managed links, and exposes a managed bucket and broken-link
  flag. Agent B may use them for explanation and the availability matrix.
  Agent B must not depend on Agent A's mutable switching code.
- Agent A's `(*project).evaluateCurrent(selected []scope) (readiness, error)`
  in `internal/cli/readiness.go` is read-only. Its result carries the branch,
  `target`, whether resolution succeeded, each selected scope's
  directory/link/expected-file facts, and per-scope readiness problems. A
  missing managed link is unhealthy
  for `check` but repairable by `apply` when its target exists. A real `.env`
  or foreign symlink is an apply blocker. A broken managed link may be
  repointed to an available expected target. `apply` plans from these facts,
  reports changed, unchanged, and failed counts, and returns nonzero when
  incomplete. Dry-run uses the same plan without writes. These readiness and
  apply types remain Agent A-owned internals; Agent B's commands do not need
  to import them.
- Agent B's bulk bucket and scaffold commands should return ordinary `error`
  values to their handlers, report each scope's result, and use existing
  `blocked`, `usage`, `envErr`, and `configErr` classes as appropriate. A
  partial result returns nonzero; successful and already-present scopes remain
  reported. No switching helper is required by Agent B's implementation.
- Agent B owns parsing and command-specific help for `map`, `bucket`, `init`,
  and `scope` in its files, and hands any top-level command/help text to Agent
  A. Agent A owns `internal/cli/cli.go` and top-level registration/help.
  Command-specific `--help` must exit zero before project/config loading.

Exit codes remain 0 success, 1 blocked or incomplete, 2 invalid/missing
configuration, 3 usage, and 4 environment failure. The internal hook always
returns zero. All command output is ASCII and env file contents are opaque.

Agent B should import Agent A's T00 interface commit into its worktree before
dependent work and acknowledge the contract. Changes to these shared
interfaces require a message and agreement between the two agents.
