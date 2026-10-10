# Status and branch mappings

`envbuckets status` shows the branch (or detached HEAD), the bucket the links
point to, and how it relates to the branch's rules. It reports missing, broken,
foreign, and replaced links, links Git does not ignore, and a missing hook block.
Run `envbuckets switch` to repair links after resolving any files in the way.
Run `envbuckets init` to reinstall a hook overwritten by another tool.

```sh
envbuckets status
envbuckets branches                     # current branch and rule-matched branches
envbuckets branches main release/2.0    # check names, including future branches
envbuckets branches 'release/**'        # matching local branches
envbuckets branches --bucket dev        # all branches mapped to dev
envbuckets branches '**'                # every local branch
```

Name arguments keep their order; patterns expand to sorted local branches.
Overlapping selectors list each branch once. `--bucket` filters the configured
mapping, so a missing production bucket remains under `--bucket prod` even when
checkouts fall back to dev. A missing default means checkouts preserve the links.

Both commands write their report to stdout and change nothing. Diagnostics use
stderr. They inspect managed file metadata and link targets without opening file
contents. Column alignment handles wide and combining characters in branch names.

In a terminal, the current branch in `branches` and the bucket in use in `status`
are green. Problem paths are red. Diagnostic prefixes `fatal:` and `error:` are
red; `warning:` and `hint:` are yellow. Piped or redirected streams stay plain,
and each stream is detected separately. Set `NO_COLOR` to any non-empty value
(including `yes`, `false`, or `0`) to disable color. `TERM=dumb` also disables it.
These settings apply to checkout hooks too; all message text stays the same.

Windows console output enables virtual terminal processing for the command and
restores the previous mode afterwards. Consoles that cannot enable it get plain
text. Git Bash and mintty terminal pipes support color without console mode changes.

`task bench` measures the real CLI's branch listing and checkout hook with 1 and
1000 local branches. Setup uses one `git update-ref --stdin` call and is excluded
from the timings. Hooks do not enumerate branches. Short Go test runs skip these
benchmarks; EB-07 will establish release budgets across runners.
