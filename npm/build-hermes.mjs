import { execFileSync } from "node:child_process";
import { cpSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = join(HERE, "..");
const STAGE = join(HERE, ".stage-hermes");

const TARGETS = [
  { node: "darwin-arm64", goos: "darwin", goarch: "arm64" },
  { node: "darwin-x64",   goos: "darwin", goarch: "amd64" },
  { node: "linux-arm64",  goos: "linux",  goarch: "arm64" },
  { node: "linux-x64",    goos: "linux",  goarch: "amd64" },
  { node: "win32-arm64",  goos: "windows",goarch: "arm64" },
  { node: "win32-x64",    goos: "windows",goarch: "amd64" },
];

function run(cmd, args, opts) {
  try {
    return execFileSync(cmd, args, { stdio: "pipe", encoding: "utf8", ...opts });
  } catch (e) {
    console.error(`${cmd} ${args.join(" ")} failed (exit ${e.status})`);
    if (e.stderr) console.error(e.stderr.trim());
    if (e.stdout) console.error(e.stdout.trim());
    throw e;
  }
}

const tag = process.argv[2] ?? process.env.GITHUB_REF_NAME;
if (!tag) {
  console.error("usage: node npm/build-hermes.mjs <tag> [--publish] [--dry-run]");
  process.exit(1);
}
const version = tag.replace(/^(hermes-)?npm-v/, "").replace(/^v/, "");
const doPublish = process.argv.includes("--publish");

// --check-auth: verify npm credentials before building
if (process.argv.includes("--check-auth")) {
  const token = process.env.NPM_TOKEN || process.env.NODE_AUTH_TOKEN;
  if (!token) {
    console.error("No NPM_TOKEN or NODE_AUTH_TOKEN in environment. Set NPM_TOKEN=<token> and re-run.");
    process.exit(1);
  }
  // Use a temp npmrc to avoid ~/.npmrc permission issues
  const tmpDir = join(ROOT, "npm", ".stage-hermes-check");
  mkdirSync(tmpDir, { recursive: true });
  const npmrc = join(tmpDir, ".npmrc");
  writeFileSync(npmrc, `//registry.npmjs.org/:_authToken=${token}\n`);
  const env = { ...process.env, npm_config_userconfig: npmrc };

  console.log("Checking npm auth...");
  try {
    const who = run("npm", ["whoami"], { env }).trim();
    console.log(`  Authenticated as: ${who}`);
    console.log("  Token OK");
  } catch {
    console.error("  Token is INVALID. Create one at https://www.npmjs.com/settings/aliatx2017/tokens");
    console.error("  Token type: Automation, packages & scopes: Read and write");
    process.exit(1);
  }
  // Check each package exists
  for (const t of TARGETS) {
    const name = `@aliatx2017/reasonix-hermes-${t.node}`;
    try {
      run("npm", ["view", name, "version"], { env });
      console.log(`  ${name}: exists`);
    } catch {
      console.log(`  ${name}: will be created on first publish`);
    }
  }
  console.log("  reasonix-hermes: main package (will be created/updated on publish)");
  rmSync(tmpDir, { recursive: true, force: true });
  process.exit(0);
}

rmSync(STAGE, { recursive: true, force: true });
mkdirSync(STAGE, { recursive: true });

const subPackages = [];
for (const t of TARGETS) {
  const name = `@aliatx2017/reasonix-hermes-${t.node}`;
  const dir = join(STAGE, `hermes-${t.node}`);
  const exe = t.goos === "windows" ? "reasonix.exe" : "reasonix";
  mkdirSync(join(dir, "bin"), { recursive: true });

  console.log(`build ${t.goos}/${t.goarch} -> ${name}`);
  run("go", [
    "build", "-trimpath", "-ldflags", `-s -w -X main.version=${tag}`,
    "-o", join(dir, "bin", exe), "./cmd/reasonix",
  ], {
    cwd: ROOT,
    stdio: "inherit",
    env: { ...process.env, CGO_ENABLED: "0", GOOS: t.goos, GOARCH: t.goarch },
  });

  writeFileSync(join(dir, "package.json"), `${JSON.stringify({
    name, version,
    description: `Reasonix-Hermes prebuilt binary for ${t.node}.`,
    os: [t.goos === "windows" ? "win32" : t.goos],
    cpu: [t.goarch === "amd64" ? "x64" : "arm64"],
    files: ["bin/"], license: "MIT",
    repository: { type: "git", url: "git+https://github.com/aliatx2017/reasonix-hermes.git" },
  }, null, 2)}\n`);
  subPackages.push({ name, dir });
}

const mainDir = join(STAGE, "reasonix-hermes");
mkdirSync(mainDir, { recursive: true });
cpSync(join(HERE, "hermes", "bin"), join(mainDir, "bin"), { recursive: true });
cpSync(join(ROOT, "README.md"), join(mainDir, "README.md"));

const mainPkg = JSON.parse(readFileSync(join(HERE, "hermes", "package.json"), "utf8"));
mainPkg.version = version;
for (const key of Object.keys(mainPkg.optionalDependencies))
  mainPkg.optionalDependencies[key] = version;
writeFileSync(join(mainDir, "package.json"), `${JSON.stringify(mainPkg, null, 2)}\n`);

if (!doPublish) {
  console.log(`\nstaged ${version} in ${STAGE} (pass --publish to publish, --check-auth to verify token)`);
  process.exit(0);
}

// Check auth before attempting publish
console.log("Checking npm auth before publish...");
const npmToken = process.env.NPM_TOKEN || process.env.NODE_AUTH_TOKEN;
if (!npmToken) {
  console.error("  No NPM_TOKEN or NODE_AUTH_TOKEN in environment.");
  console.error("  Set NPM_TOKEN=<token> and re-run.");
  process.exit(1);
}

// Write a temporary .npmrc so npm in subdirectories finds auth
const npmrcPath = join(STAGE, ".npmrc");
writeFileSync(npmrcPath, `//registry.npmjs.org/:_authToken=${npmToken}\n`);

const npmEnv = { ...process.env, npm_config_userconfig: npmrcPath };

try {
  const who = run("npm", ["whoami"], { env: npmEnv }).trim();
  console.log(`  Authenticated as: ${who}`);
} catch {
  console.error("  Token is invalid. Check https://www.npmjs.com/settings/aliatx2017/tokens");
  process.exit(1);
}

const publishArgs = ["publish", "--access", "public"];
const otp = process.env.NPM_OTP;
if (otp) publishArgs.push("--otp", otp);

let failed = false;
for (const sub of subPackages) {
  console.log(`publish ${sub.name}@${version}`);
  try {
    run("npm", publishArgs, { cwd: sub.dir, env: npmEnv, stdio: "inherit" });
  } catch {
    console.error(`  FAILED: ${sub.name}. Check token permissions at npmjs.com.`);
    console.error("  Token must have: Automation type, Packages & scopes → Read and write");
    failed = true;
    break;
  }
}

if (!failed) {
  console.log(`publish reasonix-hermes@${version}`);
  try {
    run("npm", publishArgs, { cwd: mainDir, env: npmEnv, stdio: "inherit" });
  } catch {
    console.error("  FAILED: reasonix-hermes (main package)");
    failed = true;
  }
}

if (failed) process.exit(1);
console.log(`\nPublished ${version} — all 7 packages OK.`);

