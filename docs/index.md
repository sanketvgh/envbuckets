# envbuckets user guides

Start with the [project README](../README.md) for installation and a quick
walkthrough. These guides explain setup, daily use, and recovery:

- [Setup and adding files](setup.md): initialize a project, prepare a fresh
  clone, add other local files, and create an environment.
- [Monorepos](monorepos.md): keep app and package files at their usual paths and
  switch them together with one branch mapping.
- [Switching buckets](switch.md): branch rules, manual switches, checkout hooks,
  and Windows requirements.
- [Status and branch mappings](status.md): inspect the active files, check branch
  mappings, and control terminal color.
- [Troubleshooting](troubleshooting.md): resolve import errors, blocked links,
  missing buckets, and hook problems.
- [Leaving envbuckets](uninstall.md): restore active files, keep other buckets,
  and remove the CLI.

The CLI works offline and never opens or prints managed file contents. Keep
private values out of Git and issue reports. Contributors can find checks and
the product contract in [developer documentation](https://github.com/sanketvgh/envbuckets/blob/main/docs-eb/DEVELOPMENT.md)
and [PRODUCT.md](https://github.com/sanketvgh/envbuckets/blob/main/docs-eb/PRODUCT.md).
