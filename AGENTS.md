# Working on envbuckets

envbuckets is a Go CLI that selects project-local `.env.d/<bucket>/.env` files
through `.env` symlinks as Git branches change. Start with [README.md](README.md)
and the relevant page in [docs/](docs/README.md); use
[docs/SPEC.md](docs/SPEC.md) for the detailed behavior contract.

- Run `pnpm install` once per clone to install the Markdown formatter.
- Run `task check` before submitting changes. It runs formatting/lint (Go
  with golangci-lint, Markdown with `oxfmt`; `task fmt` rewrites both), unit
  tests, and the real-Git integration scripts. The scripts need symlink support;
  use Linux or WSL if your Windows session cannot create symlinks.
- Keep the product value-blind: never add code that reads, parses, logs, or
  prints real environment-file contents. Tests may use synthetic values.
- Keep bucket and scope operations inside the repository. Do not follow
  symlinked scope directories, bucket directories, or bucket files outside it.
- Keep the checkout hook nonblocking. It exits 0, leaves unsafe or missing
  links alone, and lets other scopes proceed.
- Human CLI output is concise and may change. Agents and scripts should use
  `--json`, documented in [docs/agent-integration.md](docs/agent-integration.md).
  Preserve JSON field names, issue codes, and exit codes within schema version 1.
- When changing command behavior or output, update relevant unit tests,
  `testdata/script/*.txtar`, and the affected documentation. Do not reintroduce
  `next:` labels or machine error classes in human-facing messages.
- Use `<gitmoji> <type>(<scope>): <summary>` for commits and PR titles, such as
  `✨ feat(cli): add readiness checks`. Keep the type and scope meaningful for
  the final change.
- Run `task security` after changing dependencies or anything under
  `.github/`. It runs `govulncheck` and `zizmor`; CI runs both too. Pin every
  new GitHub Action to a commit SHA with the version in a trailing comment,
  and do not add caches to the release job.
- Run `task hooks` once per clone. It installs a pre-push hook that runs lint,
  tests, and `task security`, so a failing push is stopped locally.
- `main` is protected by the "main gate" ruleset: changes land through a pull
  request, and `ci`, `security`, `integration`, `analyze` (CodeQL), and
  `review` (dependency review) must pass first. Nobody can bypass it.
- Use `task playground` after `task snapshot` to try the packed npm package in
  the gitignored `playground/` directory. Run `git init` there yourself if you
  need a repository; never rely on the parent repo.
- Release publishing is handled by `.github/workflows/release.yml`. `task
snapshot` builds release artifacts locally without publishing.
