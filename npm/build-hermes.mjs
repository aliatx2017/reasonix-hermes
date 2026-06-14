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

const tag = process.argv[2] ?? process.env.GITHUB_REF_NAME;
if (!tag) {
  console.error("usage: node npm/build-hermes.mjs <tag>   (e.g. v1.7.0)");
  process.exit(1);
}
const version = tag.replace(/^(hermes-)?npm-v/, "");
const publish = process.argv.includes("--publish");

rmSync(STAGE, { recursive: true, force: true });
mkdirSync(STAGE, { recursive: true });

const subPackages = [];
for (const t of TARGETS) {
  const name = `@aliatx2017/reasonix-hermes-${t.node}`;
  const dir = join(STAGE, `hermes-${t.node}`);
  const exe = t.goos === "windows" ? "reasonix.exe" : "reasonix";
  mkdirSync(join(dir, "bin"), { recursive: true });

  console.log(`build ${t.goos}/${t.goarch} -> ${name}`);
  execFileSync(
    "go",
    [
      "build",
      "-trimpath",
      "-ldflags",
      `-s -w -X main.version=${tag}`,
      "-o",
      join(dir, "bin", exe),
      "./cmd/reasonix",
    ],
    {
      cwd: ROOT,
      stdio: "inherit",
      env: { ...process.env, CGO_ENABLED: "0", GOOS: t.goos, GOARCH: t.goarch },
    },
  );

  writeFileSync(
    join(dir, "package.json"),
    `${JSON.stringify(
      {
        name,
        version,
        description: `Reasonix-Hermes prebuilt binary for ${t.node}.`,
        os: [t.goos === "windows" ? "win32" : t.goos],
        cpu: [t.goarch === "amd64" ? "x64" : "arm64"],
        files: ["bin/"],
        license: "MIT",
        repository: {
          type: "git",
          url: "git+https://github.com/aliatx2017/reasonix-hermes.git",
        },
      },
      null,
      2,
    )}\n`,
  );
  subPackages.push({ name, dir });
}

const mainDir = join(STAGE, "reasonix-hermes");
mkdirSync(mainDir, { recursive: true });
cpSync(join(HERE, "hermes", "bin"), join(mainDir, "bin"), { recursive: true });
cpSync(join(ROOT, "README.md"), join(mainDir, "README.md"));

const mainPkg = JSON.parse(
  readFileSync(join(HERE, "hermes", "package.json"), "utf8"),
);
mainPkg.version = version;
for (const key of Object.keys(mainPkg.optionalDependencies)) {
  mainPkg.optionalDependencies[key] = version;
}
writeFileSync(
  join(mainDir, "package.json"),
  `${JSON.stringify(mainPkg, null, 2)}\n`,
);

if (!publish) {
  console.log(`\nstaged ${version} in ${STAGE} (dry run; pass --publish to publish)`);
  process.exit(0);
}

const publishArgs = ["publish", "--access", "public"];
for (const sub of subPackages) {
  console.log(`publish ${sub.name}@${version}`);
  execFileSync("npm", publishArgs, { cwd: sub.dir, stdio: "inherit" });
}
console.log(`publish reasonix-hermes@${version}`);
execFileSync("npm", publishArgs, { cwd: mainDir, stdio: "inherit" });
