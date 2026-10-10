# Monorepos

Use one `.envbuckets.json` at the Git repository root. A branch mapping chooses
one bucket for the whole repository. Each app keeps its usual file paths;
those paths are repeated inside the bucket.

```text
monorepo/                         .env.d/dev/
|-- .envbuckets.json               `-- apps/
|-- .gitignore                        |-- api/
`-- apps/                             |   |-- .env
    |-- api/                          |   `-- service-account.json
    |   |-- .env                      `-- web/
    |   `-- service-account.json           `-- .env.local
    `-- web/
        `-- .env.local

Working files become links        Bucket files hold your values
```

For example, the link at `apps/api/.env` points to
`../../.env.d/dev/apps/api/.env`. Both API and web files use `dev` on that branch.
Editing either app's linked file edits its file in the active bucket.

## Set up the repository

Start from the repository root with the apps' local files in place:

```sh
envbuckets init -n
envbuckets init
envbuckets add apps/api/service-account.json
envbuckets status
```

`init` finds untracked `.env` and `.env.*` files in app and package directories.
It skips tracked examples and ignored directories such as `node_modules/`.
Use `add` for other local files, such as service accounts or settings JSON.
Create those files before running `add`.

Commit the root `.envbuckets.json` and `.gitignore`. Keep `.env.d/` and the
working links ignored. After cloning, each teammate runs `init`, creates their
own local files from tracked examples, then adds them. If their branch maps to
a missing bucket and no files are linked yet, they first create that bucket's
directory under `.env.d/`. See [setup](setup.md#fresh-clone).

## Create environments and branch rules

Before adding branch rules, create the buckets you need:

```sh
envbuckets switch -c staging
# Fill in staging values through the apps' working paths.
envbuckets switch
envbuckets switch -c prod
# Fill in production values through the apps' working paths.
envbuckets switch
```

Each new bucket starts with empty files at the current bucket's paths. Values
are never copied between environments. Then edit the root config:

```json
{
  "$schema": "https://raw.githubusercontent.com/sanketvgh/envbuckets/main/schema/envbuckets.schema.json",
  "default": "dev",
  "rules": [
    { "branch": "main", "bucket": "staging" },
    { "branch": "release/**", "bucket": "prod" }
  ]
}
```

Run `envbuckets switch` to apply the rules to your current branch. Feature
branches use `dev`, `main` uses `staging`, and release branches use `prod`.
Checking out a branch switches every managed app file to that bucket.

## Files can differ between buckets

A bucket doesn't need every file. If the web app has no production `.env.local`,
switching to `prod` removes its working link and keeps its `dev` file. Preview
this with `envbuckets switch -n prod`. To add a production-only file, switch to
`prod`, create the file at its normal project path, then run `add` for that path.

Buckets group files across the whole repo. To test a combination such as a
local API with staging web settings, create a bucket containing that combination
and select it with `envbuckets switch <bucket>`. Plain `switch` returns to the
branch's bucket. See [switching](switch.md) and [troubleshooting](troubleshooting.md).
