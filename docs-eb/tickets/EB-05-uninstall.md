# EB-05: uninstall

**Goal:** leave the project working without envbuckets, with no data lost.

**Scope:** find managed paths from bucket contents without reading the config, replace each envbuckets link with a real copy of its target (temp file next to it, then atomic rename), then remove the hook block. Keep `.env.d/`, the config, and the ignore block. `uninstall -n` lists the files it would turn back into real files and the hook change, then stops.

## Acceptance criteria

- Each formerly linked path is a real file with the active bucket's bytes, and bucket files are unchanged.
- Broken links are reported and left alone; the rest completes.
- Only the envbuckets hook block is removed. Other hook content stays byte-identical and executable, and a hook file that held only our block is deleted.
- Works with a missing or broken config.
- Running it twice, or on a repo that was never set up, does nothing and exits 0.
- If interrupted, the project still runs and rerunning finishes the job.
- `uninstall -n` changes nothing and exits with the code a real run would return.
