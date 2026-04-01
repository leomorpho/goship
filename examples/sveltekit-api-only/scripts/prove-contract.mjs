import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { spawn } from "node:child_process";

const exampleDir = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..");
const repoRoot = path.resolve(exampleDir, "..", "..");
const generatedContractPath = path.join(exampleDir, "generated", "goship-contract.json");

function run(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: options.cwd ?? repoRoot,
      env: { ...process.env, ...(options.env ?? {}) },
      stdio: ["ignore", "pipe", "pipe"],
    });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk) => (stdout += chunk));
    child.stderr.on("data", (chunk) => (stderr += chunk));
    child.on("error", reject);
    child.on("close", (code) => {
      if (code === 0) {
        resolve({ stdout, stderr });
      } else {
        reject(new Error(`${command} ${args.join(" ")} failed with code ${code}\n${stdout}\n${stderr}`));
      }
    });
  });
}

async function waitFor(url, timeoutMs = 10000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url);
      if (res.ok) return res;
    } catch {}
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error(`timed out waiting for ${url}`);
}

function reservePort() {
  const port = 4600 + Math.floor(Math.random() * 1000);
  return String(port);
}

async function main() {
  const contract = JSON.parse(await readFile(generatedContractPath, "utf8"));
  if (contract.contract_version !== "api-only-same-origin-sveltekit-v1") {
    throw new Error(`unexpected contract version: ${contract.contract_version}`);
  }
  if (contract.browser_contract?.authMode !== "same-origin auth/session") {
    throw new Error("missing same-origin auth metadata");
  }
  const statusRoute = contract.routes.find((route) => route.path === "/api/v1/status");
  if (!statusRoute) {
    throw new Error("generated contract missing /api/v1/status");
  }

  const work = await mkdtemp(path.join(tmpdir(), "goship-sveltekit-proof-"));
  const shipbin = path.join(work, "ship");
  try {
    await run("go", ["build", "-o", shipbin, "./tools/cli/ship/cmd/ship"], { cwd: repoRoot });
    await run(shipbin, ["new", "demo", "--module", "example.com/demo", "--api", "--no-i18n"], { cwd: work });
    const appDir = path.join(work, "demo");
    await run(shipbin, ["db:migrate"], { cwd: appDir });

    const port = reservePort();
    const web = spawn("go", ["run", "./cmd/web"], {
      cwd: appDir,
      env: { ...process.env, PORT: port },
      stdio: ["ignore", "ignore", "ignore"],
    });

    try {
      const res = await waitFor(`http://127.0.0.1:${port}${statusRoute.path}`);
      const payload = await res.json();
      if (payload?.data?.status !== "ok") {
        throw new Error(`unexpected API payload: ${JSON.stringify(payload)}`);
      }
    } finally {
      web.kill("SIGKILL");
      await new Promise((resolve) => web.once("close", resolve));
    }
  } finally {
    await rm(work, { recursive: true, force: true });
  }
}

await main();
