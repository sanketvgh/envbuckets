#!/usr/bin/env node
"use strict";

const { spawnSync } = require("node:child_process");
const path = require("node:path");

const pkg = `@envbuckets/${process.platform}-${process.arch}`;
const exe = process.platform === "win32" ? "envbuckets.exe" : "envbuckets";

let bin;
try {
  bin = path.join(path.dirname(require.resolve(`${pkg}/package.json`)), exe);
} catch {
  console.error(
    `envbuckets: no prebuilt binary for ${process.platform}-${process.arch}.\n` +
      `Expected optional dependency "${pkg}". If you installed with --omit=optional, reinstall without it.`,
  );
  process.exit(4);
}

const result = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(`envbuckets: failed to start ${bin}: ${result.error.message}`);
  process.exit(4);
}
process.exit(result.status ?? 1);
