# envbuckets

**Stop copy-pasting `.env` files every time you switch branches.**

[![ci](https://github.com/sanketvgh/envbuckets/actions/workflows/ci.yml/badge.svg)](https://github.com/sanketvgh/envbuckets/actions/workflows/ci.yml)
[![npm](https://img.shields.io/npm/v/envbuckets?label=npm)](https://www.npmjs.com/package/envbuckets)
[![license](https://img.shields.io/github/license/sanketvgh/envbuckets)](LICENSE)

> [!WARNING]
> **Early alpha, under active development.** The CLI is not implemented yet.
> Commands and config may change without notice until v1.0.

envbuckets keeps one `.env` per bucket (`dev`, `staging`, `prod`) and points
`.env` at the right one for the branch you're on. A git hook does the switch
on every checkout. It never reads what's inside your env files.

```text
.env  →  .env.d/staging/.env      # on main
.env  →  .env.d/prod/.env         # on release/*
```

## Install

```sh
npm install -g envbuckets
```

Windows, macOS, Linux. Node 18+. No Go required.

## Usage

```sh
envbuckets init
envbuckets bucket add staging
envbuckets map add main staging
envbuckets map add "release/*" prod
git checkout release/1.2        # .env now → .env.d/prod/.env
envbuckets status               # which env am I on?
```

Monorepo? `envbuckets scope add apps/api` gives each service its own
`.env.d/` while rules stay shared.

## License

[MIT](LICENSE)
