# Working on envbuckets

envbuckets is being rebuilt. Read [docs-eb/PRODUCT.md](docs-eb/PRODUCT.md) and
the relevant implementation ticket in [docs-eb/tickets/](docs-eb/tickets/README.md).

- Run `task check` before submitting changes. It runs formatting/lint, unit
  tests, and the real-Git integration scripts. The scripts need symlink support;
  use Linux or WSL if your Windows session cannot create symlinks.
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
