# EB-07: Release and acceptance

**Goal:** ship the new product as one coherent CLI.

**Scope:** README and command help for the new product; remove scope-era code, commands, and docs, and the TOML dependency; update `AGENTS.md` and `CLAUDE.md`; end-to-end scripts for every acceptance criterion in `PRODUCT.md`; npm package smoke test.

## Acceptance criteria

- Every acceptance criterion in `PRODUCT.md` has an automated test or a documented platform limit.
- Every sample run in `PRODUCT.md` matches real output, checked by an integration script.
- `task check` and `task security` pass.
- The packed npm package works in the playground from its own Git repository.
