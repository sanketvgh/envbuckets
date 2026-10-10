# Release acceptance coverage

This maps the numbered criteria in [PRODUCT.md](PRODUCT.md) to automated checks.
Tests read only synthetic managed files. A coverage entry identifies a check;
passing same-code CI is still required before EB-07 can be closed.

| Criterion | Automated evidence |
| --- | --- |
| 1. Import local files unchanged, preserve tracked examples, ignore managed paths | `init-add.txtar`, `acceptance-walkthrough.txtar`, `TestInitImportsAndReruns` |
| 2. Rule-matched and default branch checkout | `checkout.txtar`, `acceptance-team.txtar`, `acceptance-walkthrough.txtar` |
| 3. Schema/CLI agreement, documented configs, `$schema`, offline validation | `internal/config/schema_test.go`, schema invalid fixtures, `acceptance-offline.txtar`; `TestInitImportsAndReruns` checks generated `$schema` |
| 4. Every Git pattern row and malformed patterns | `internal/pattern/pattern_test.go`, recorded Git fixture and its regeneration test, config tests, `acceptance-rules.txtar` |
| 5. Branch selection, filters, future names, temporary bucket, 1000 refs with one enumeration | `reports.txtar`, CLI report tests, `acceptance-scale.txtar` (1000 lines and one traced `for-each-ref`), `internal/gitx/branches_test.go` |
| 6. Remove working links absent from target, retain bucket files | `switch.txtar`, `acceptance-walkthrough.txtar`, `TestSwitchTemporaryAndRepair` |
| 7. Preserve real files, switch healthy paths, report obstruction | `checkout.txtar`, `switch.txtar`, `acceptance-walkthrough.txtar`, CLI status/switch tests |
| 8. Safe checkout on missing bucket/config/binary, detached HEAD, file checkout; fallback from prod | `checkout.txtar`, `hooks.txtar`, `acceptance-clone.txtar`, switch/hook unit tests |
| 9. Create empty bucket files and reset manual overrides | `init-add.txtar`, `checkout.txtar`, `acceptance-walkthrough.txtar`, `TestCreateBucketAndDryRun` |
| 10. Restore files, keep other buckets/custom hooks, reinitialize | `uninstall.txtar`, `uninstall-empty.txtar`, CLI uninstall tests, packed npm smoke |
| 11. No secret marker output and offline operation | `acceptance-offline.txtar` runs every command with unreachable schema/proxy URLs; exact documentation-output checks and npm smoke reject synthetic secret markers |
| 12. Unreadable managed files | `init-add-unreadable.txtar`, `switch-unreadable.txtar`, `reports-unreadable.txtar`, `uninstall-unreadable.txtar` |
| 13. Dry-run plans, byte-identical repositories, same planned exit codes | CLI import/switch/uninstall snapshot tests and txtar scripts; npm smoke hashes the entire synthetic repository for all four dry runs |
| 14. Terminal-only color and identical text | `colors.txtar`, `internal/output` terminal/stream tests, CLI terminal tests on Windows |
| 15. Prefixes, silence, dry runs, report layout, stdout/stderr | Exact comparisons in `reports.txtar`, `colors.txtar`, `uninstall.txtar`, CLI tests, and the documentation scripts |

`acceptance-rules`, `acceptance-scale`, `acceptance-walkthrough`, `acceptance-clone`,
and `acceptance-team` check every PRODUCT.md CLI sample against its actual text.
`TestAcceptanceSampleCoverage` verifies that every transcript has an immediately
preceding matching command in a script. Only fixture path, CRLF, tab presentation,
and code-fence indentation are normalized. For Git commands, only envbuckets hook
lines are compared; Git's version-dependent checkout messages and abbreviated
commit IDs are outside the CLI contract. Silent branch checkouts are also asserted.

## Platform limits

- Windows integration and npm smoke require Developer Mode or symlink privilege.
  This research session lacks that privilege. The suite does not blanket-skip
  Windows; CI must verify the same code before release.
- `chmod 000` cannot represent Unix unreadability on Windows. The four permission
  scripts skip with an explicit reason and run on Linux/macOS.
- Git LFS hook coexistence is checked when `git-lfs` is installed; the script
  names that dependency when skipping.
- Rename on Windows uses one Go `MoveFileEx(MOVEFILE_REPLACE_EXISTING)` call, with
  no envbuckets delete step. Microsoft does not document it as atomic. Rerunning
  `switch` converges after an interruption; Unix rename is atomic per path,
  while a whole multi-file switch is not a transaction.
- Color tests need a real terminal. Automated Windows console checks and the
  Unix PTY tests cover terminal behavior; EB-06 tracks its remaining manual
  Windows Terminal visual review, deferred by the user until after release.

See [developer checks](DEVELOPMENT.md) for commands and EB-07 for measured
benchmark budgets, local results, CI evidence, and release blockers.

## Shipped documentation

Public guides live in [`../docs/`](../docs/index.md) with lowercase slug
filenames. `TestPackageDocumentation` checks the generated umbrella README
and both package licenses using synthetic source files. Packed npm smoke
checks both installed licenses and the installed README against repository
sources before exercising the CLI. Binary snapshot archives include the
README, license, and all user-guide files; EB-10 records archive inspection
and final documentation-link evidence.
