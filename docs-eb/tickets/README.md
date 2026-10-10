# Implementation tickets

These tickets split [`../PRODUCT.md`](../PRODUCT.md) into reviewable pieces. Each ticket ships with unit tests, txtar integration scripts where Git behavior matters, and doc updates.

| ID    | Ticket                                                  | Depends on     | Status                                              |
| ----- | ------------------------------------------------------- | -------------- | --------------------------------------------------- |
| EB-00 | [Clear the alpha code (pre-work)](EB-00-clear-alpha.md) | none           | Passed (Actions run `37812978741` on `7bc8403`)    |
| EB-01 | [Paths, buckets, and config](EB-01-paths-config.md)     | EB-00          | Passed (Actions run `37946531174` on `d74be56`)     |
| EB-02 | [Switch engine and hook](EB-02-switch-hook.md)          | EB-01          | Passed (Actions run `37951292646` on `f1c2133`; local/CI verification accepted) |
| EB-03 | [init, add, and switch -c](EB-03-init-add.md)           | EB-01, EB-02   | Passed (Actions run `37960285426` on `823e0d9`; local/CI verification accepted) |
| EB-04 | [status and branches](EB-04-status-branches.md)         | EB-01, EB-02   | Passed (Actions run `38020427583` on `bbef602`; local/CI verification accepted) |
| EB-05 | [uninstall](EB-05-uninstall.md)                         | EB-01, EB-02   | Passed (Actions run `38022429016` on `bcc6638`; local/CI verification accepted) |
| EB-06 | [Colored output with Lip Gloss](EB-06-colors.md)        | EB-01, EB-04   | CI passed (run `38024972163` on `a50e2cf`); user deferred Windows Terminal review until after release |
| EB-07 | [Release and acceptance](EB-07-release.md)              | EB-01 to EB-06 | CI passed (run `38027455901` on `e211da4`); schema on main pending |
| EB-08 | [Earlier lint and format safeguards](EB-08-lint-safeguards.md) | EB-07   | Done (Actions run `37954669747` on `fcdfa8c`) |
| EB-09 | [Stricter bug-catching checks](EB-09-strict-checks.md) | EB-08 | Passed (Actions run `37966549169` on `8d49af3`; local/CI verification accepted) |
| EB-10 | [Final documentation and pre-PR gate](EB-10-pre-pr-gate.md) | EB-00 to EB-09 | Local docs/packaging verified; final same-code CI pending |

Each ticket ends with an exit checklist. Move to the next ticket only when every box in the previous one is ticked.

EB-10 audits the final PR readiness and carries forward unresolved earlier gates.
EB-07's schema availability on remote `main` is verified after merge and remains
a release blocker; it does not block opening the PR that adds the schema.
EB-00's historical checklist discrepancy and the user-deferred EB-06 visual
review are recorded in EB-10; required automated verification remains in place.
