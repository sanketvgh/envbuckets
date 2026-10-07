# Implementation tickets

These tickets split [`../PRODUCT.md`](../PRODUCT.md) into reviewable pieces. Each ticket ships with unit tests, txtar integration scripts where Git behavior matters, and doc updates.

| ID    | Ticket                                                  | Depends on     |
| ----- | ------------------------------------------------------- | -------------- |
| EB-00 | [Clear the alpha code (pre-work)](EB-00-clear-alpha.md) | none           |
| EB-01 | [Paths, buckets, and config](EB-01-paths-config.md)     | EB-00          |
| EB-02 | [Switch engine and hook](EB-02-switch-hook.md)          | EB-01          |
| EB-03 | [init, add, and switch -c](EB-03-init-add.md)           | EB-01, EB-02   |
| EB-04 | [status and branches](EB-04-status-branches.md)         | EB-01, EB-02   |
| EB-05 | [uninstall](EB-05-uninstall.md)                         | EB-01, EB-02   |
| EB-06 | [Colored output with Lip Gloss](EB-06-colors.md)        | EB-01, EB-04   |
| EB-07 | [Release and acceptance](EB-07-release.md)              | EB-01 to EB-06 |
