# Configuration and branch rules

`.envbuckets.toml` is the shared, schema-versioned configuration at the
repository root. It records branch-to-bucket rules and registered scopes,
never env values. Use the CLI to edit it; an unreadable config makes normal
commands stop instead of guessing. `uninstall` can still deactivate the
project if the config is broken.

## Match branches

Rules run in order. The first matching pattern wins. A `*` rule stays last.
Like Git globs, `*` and `?` stop at `/`; `**` can cross directory separators
when it appears at a pattern boundary (`**/`, `/**`, or by itself). A trailing
`/` matches everything beneath that prefix. Character sets and ranges such as
`[0-9]`, negated sets such as `[!a-c]`, POSIX classes such as `[[:digit:]]`,
and backslash escapes are supported. Brace expansion is not. A local branch
pin, described in
[Branch switching](branch-switching.md), overrides every rule.

```sh
envbuckets bucket add dev
envbuckets bucket add prod
envbuckets map add 'release/*' prod
envbuckets map add '*' dev
envbuckets map list
envbuckets map explain release/2.0
```

Quote patterns so the shell does not expand `*`. `map add` requires its
bucket to exist in at least one scope. A new specific rule is inserted
before the catch-all, even when the catch-all was added earlier. A rule
selecting a bucket does not guarantee that every scope has that bucket's
`.env`; use `bucket list --all` to see gaps.

## Edit priority and targets

```sh
envbuckets map update 'release/*' staging
envbuckets map move 'release/hotfix-*' --before 'release/*'
envbuckets map move 'release/*' --after 'main'
envbuckets map rm 'release/hotfix-*'
```

`update` keeps the rule's position. `move` changes priority while keeping
its bucket, and requires exactly one of `--before` or `--after`. The
catch-all cannot be moved away from the end. `rm` removes the named
pattern. `map explain <branch>` resolves a branch without checking it out,
including a local pin and per-scope bucket availability; the branch need
not exist yet.

After changing rules, commit `.envbuckets.toml`. Run `envbuckets apply`
to update the current checkout immediately; editing rules alone does not
switch `.env`.

Bucket and scope names start with a letter or digit and then use letters,
digits, `_`, or `-`. Patterns cannot be empty or contain whitespace.

See [Status and recovery](status-and-recovery.md) when a rule matches but
the checkout is not ready.
