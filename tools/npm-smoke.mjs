// Exercise tarballs, npm's launcher, and the installed checkout hook offline.
// Every managed file is synthetic and lives in a new repository under playground/.
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
function contained(relative) {
  const target = path.resolve(root, relative);
  assert(target.startsWith(`${root}${path.sep}`), `outside repository: ${target}`);
  let current = root;
  for (const part of path.relative(root, target).split(path.sep)) {
    current = path.join(current, part);
    if (fs.existsSync(current)) {
      assert(!fs.lstatSync(current).isSymbolicLink(), `unsafe symlink: ${current}`);
    }
  }
  return target;
}
const playground = contained("playground");
fs.mkdirSync(playground, { recursive: true });
const work = fs.mkdtempSync(path.join(playground, "eb07-"));
const packs = path.join(work, ".packs");
fs.mkdirSync(packs);
const home = path.join(work, ".home");
fs.mkdirSync(home);
const config = path.join(home, "gitconfig");
fs.writeFileSync(config, "[commit]\n\tgpgsign = false\n[maintenance]\n\tautoDetach = false\n");
const npmConfig = path.join(home, "npmrc");
fs.writeFileSync(npmConfig, "");
const inheritedEnv = Object.fromEntries(
  Object.entries(process.env).filter(([key]) => !/^(GIT_|npm_config_)/i.test(key)),
);
const env = {
  ...inheritedEnv,
  HOME: home,
  USERPROFILE: home,
  XDG_CONFIG_HOME: path.join(home, "xdg"),
  GIT_CONFIG_GLOBAL: config,
  GIT_CONFIG_NOSYSTEM: "1",
  GIT_TERMINAL_PROMPT: "0",
  GIT_AUTHOR_NAME: "envbuckets smoke",
  GIT_AUTHOR_EMAIL: "smoke@example.com",
  GIT_COMMITTER_NAME: "envbuckets smoke",
  GIT_COMMITTER_EMAIL: "smoke@example.com",
  GIT_AUTHOR_DATE: "2000-01-01T00:00:00+0000",
  GIT_COMMITTER_DATE: "2000-01-01T00:00:00+0000",
  npm_config_cache: path.join(home, "npm-cache"),
  npm_config_prefix: path.join(home, "npm-global"),
  npm_config_userconfig: npmConfig,
  npm_config_globalconfig: path.join(home, "global-npmrc"),
  npm_config_offline: "true",
  NO_COLOR: "1",
};

function run(command, args, cwd = work, code = 0) {
  const result = spawnSync(command, args, { cwd, env, encoding: "utf8" });
  assert.ifError(result.error);
  assert.equal(result.status, code, `${command} ${args.join(" ")}\n${result.stderr}`);
  assert(!`${result.stdout}${result.stderr}`.includes("EB07_SYNTHETIC_SECRET"));
  return result;
}

// npm-cli.js avoids invoking a .cmd file through a shell on Windows.
const npmCLI = [
  process.env.npm_execpath,
  path.join(path.dirname(process.execPath), "node_modules", "npm", "bin", "npm-cli.js"),
  path.join(
    path.dirname(process.execPath),
    "..",
    "lib",
    "node_modules",
    "npm",
    "bin",
    "npm-cli.js",
  ),
].find((candidate) => candidate && fs.existsSync(candidate));
assert(npmCLI, "npm-cli.js is unavailable; set npm_execpath to its path");
const npm = (args, cwd) => run(process.execPath, [npmCLI, ...args], cwd);
const umbrella = contained("npm/dist/envbuckets");
const platform = contained(`npm/dist/@envbuckets/${process.platform}-${process.arch}`);
const manifest = JSON.parse(fs.readFileSync(path.join(umbrella, "package.json")));
const version = manifest.version;
assert.equal(
  manifest.optionalDependencies[`@envbuckets/${process.platform}-${process.arch}`],
  version,
);
const tarballs = [platform, umbrella].map((cwd) => {
  const [packed] = JSON.parse(
    npm(["pack", "--json", "--ignore-scripts", "--pack-destination", packs], cwd).stdout,
  );
  assert(packed.files.some((file) => file.path === "LICENSE"));
  if (cwd === umbrella) {
    assert(packed.files.some((file) => file.path === "bin/envbuckets.js"));
    assert(packed.files.some((file) => file.path === "README.md"));
  }
  return path.join(packs, packed.filename);
});
fs.writeFileSync(
  path.join(work, "package.json"),
  JSON.stringify({ name: "envbuckets-smoke", version: "1.0.0", private: true }),
);
npm([
  "install",
  "--offline",
  "--ignore-scripts",
  "--no-audit",
  "--no-fund",
  "--omit=optional",
  ...tarballs,
]);
assert.equal(
  fs.readFileSync(path.join(work, "node_modules", "envbuckets", "README.md"), "utf8"),
  fs.readFileSync(path.join(root, "README.md"), "utf8"),
);
for (const name of ["envbuckets", `@envbuckets/${process.platform}-${process.arch}`]) {
  assert.equal(
    fs.readFileSync(path.join(work, "node_modules", name, "LICENSE"), "utf8"),
    fs.readFileSync(path.join(root, "LICENSE"), "utf8"),
  );
}
const launcher = path.join(work, "node_modules", "envbuckets", "bin", "envbuckets.js");
const cli = (...args) => run(process.execPath, [launcher, ...args]);
const pathKey = Object.keys(env).find((key) => key.toUpperCase() === "PATH") || "PATH";
const executablePath = env[pathKey] || "";
delete env[pathKey];
env.PATH = `${path.join(work, "node_modules", ".bin")}${path.delimiter}${executablePath}`;
run("git", ["init", "-q", "-b", "main"]);
assert.equal(path.resolve(run("git", ["rev-parse", "--show-toplevel"]).stdout.trim()), work);
assert(cli("--version").stdout.startsWith(`envbuckets ${version} (`));
run(process.execPath, [launcher, "switch", "-x"], work, 2);
fs.mkdirSync(path.join(work, "apps", "web"), { recursive: true });
const secret = "EB07_SYNTHETIC_SECRET_DEV\n";
fs.writeFileSync(path.join(work, ".env"), secret);
fs.writeFileSync(path.join(work, "apps", "web", ".env.local"), secret);
fs.writeFileSync(path.join(work, ".env.example"), "synthetic example\n");
run("git", ["add", ".env.example"]);
run("git", ["commit", "-qm", "fixture"]);

// Only this newly created synthetic repository is ever read for dry-run hashes.
function snapshot(dir = work) {
  const result = {};
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (["node_modules", ".packs", ".home"].includes(entry.name)) continue;
    const file = path.join(dir, entry.name);
    const key = path.relative(work, file);
    if (entry.isSymbolicLink()) result[key] = fs.readlinkSync(file);
    else if (entry.isDirectory()) Object.assign(result, snapshot(file));
    else result[key] = createHash("sha256").update(fs.readFileSync(file)).digest("hex");
  }
  return result;
}
let before = snapshot();
assert.match(cli("init", "-n").stdout, /Would create .envbuckets.json/);
assert.deepEqual(snapshot(), before);
cli("init");
assert(fs.lstatSync(path.join(work, ".env")).isSymbolicLink());
assert.equal(fs.readFileSync(path.join(work, ".env.d", "dev", ".env"), "utf8"), secret);
assert.match(
  run("git", ["check-ignore", ".env", "apps/web/.env.local"]).stdout,
  /apps\/web\/.env.local/,
);
fs.writeFileSync(path.join(work, "key.json"), secret);
before = snapshot();
cli("add", "-n", "key.json");
assert.deepEqual(snapshot(), before);
assert.equal(cli("add", "key.json").stdout, "");
cli("switch", "-c", "prod");
assert.equal(fs.statSync(path.join(work, ".env.d", "prod", ".env")).size, 0);
cli("switch");
fs.writeFileSync(
  path.join(work, ".envbuckets.json"),
  JSON.stringify({ default: "dev", rules: [{ branch: "release/**", bucket: "prod" }] }),
);
before = snapshot();
cli("switch", "-n", "prod");
assert.deepEqual(snapshot(), before);
assert.match(
  run("git", ["switch", "-qc", "release/1.0"]).stderr,
  /envbuckets: Switched to bucket 'prod'/,
);
assert.match(cli("status").stdout, /Using bucket 'prod'/);
assert.match(cli("branches").stdout, /rule 'release\/\*\*'/);
run("git", ["switch", "-q", "main"]);
assert.equal(fs.readFileSync(path.join(work, ".env"), "utf8"), secret);
before = snapshot();
cli("uninstall", "-n");
assert.deepEqual(snapshot(), before);
cli("uninstall");
assert(!fs.lstatSync(path.join(work, ".env")).isSymbolicLink());
assert.equal(fs.readFileSync(path.join(work, ".env"), "utf8"), secret);
assert(fs.existsSync(path.join(work, ".env.d", "prod", ".env")));
cli("init");
cli("add", "key.json");
console.log(`Packed npm smoke passed (${process.platform}/${process.arch}, ${version}) in ${work}`);
