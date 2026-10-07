# EB-07: Release and acceptance

**Goal:** ship the new product as one coherent CLI.

**Scope:** README and command help for the new product; remove scope-era code, commands, docs, txtar scripts, and the TOML dependency; rebuild the `task bench` fixture with the new commands (it still calls `bucket add` and `map add`); update `AGENTS.md` and `CLAUDE.md`; make sure `schema/envbuckets.schema.json` is on `main` so the `$schema` URL resolves, and show editor setup in the README; end-to-end scripts for every acceptance criterion in `PRODUCT.md`; npm package smoke test.

## Acceptance criteria

- Every acceptance criterion in `PRODUCT.md` has an automated test or a documented platform limit.
- Every sample run in `PRODUCT.md` matches real output, checked by an integration script.
- `task check` and `task security` pass.
- The packed npm package works in the playground from its own Git repository.
