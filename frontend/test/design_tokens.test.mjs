import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { build } from "vite";

const frontendRoot = path.resolve(import.meta.dirname, "..");
const stylesEntry = path.resolve(frontendRoot, "../styles/styles.css");
const vanillaCounterPath = path.resolve(frontendRoot, "islands/VanillaCounter.js");

test("vite build emits framework design token CSS", async () => {
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "goship-vite-build-"));

  try {
    await build({
      configFile: false,
      css: {
        postcss: path.join(frontendRoot, "postcss.config.cjs"),
      },
      root: frontendRoot,
      build: {
        outDir: tempDir,
        emptyOutDir: true,
        rollupOptions: {
          input: {
            styles_bundle: stylesEntry,
          },
          output: {
            assetFileNames: "[name][extname]",
          },
        },
        sourcemap: false,
      },
    });

    const cssFiles = findFiles(tempDir, ".css");
    assert.ok(cssFiles.length > 0, "expected vite build to emit CSS");

    const css = cssFiles.map((file) => fs.readFileSync(file, "utf8")).join("\n");
    assert.match(css, /--gs-color-background:/);
    assert.match(css, /--gs-space-6:/);
    assert.match(css, /--gs-shadow-float:/);
    assert.match(css, /\.gs-page\b/);
    assert.match(css, /\.gs-kicker\b/);
    assert.match(css, /\.gs-stack\b/);
    assert.match(css, /\.gs-card\b/);
    assert.match(css, /\.gs-nav\b/);
    assert.match(css, /\.gs-nav-item\b/);
    assert.match(css, /\.gs-layout-shell\b/);
    assert.match(css, /\.gs-layout-header\b/);
    assert.match(css, /\.gs-layout-content\b/);
    assert.match(css, /\.gs-layout-footer\b/);
    assert.match(css, /\.gs-island-mount\b/);
    assert.match(css, /\.gs-alert\b/);
    assert.match(css, /\.gs-color-muted\b/);
    assert.match(css, /\.gs-elevation-float\b/);
    assert.match(css, /\.gs-field-input\b/);
    assert.match(css, /\.gs-field-hint\b/);
    assert.match(css, /\.gs-field-error\b/);
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
});

test("vanilla counter island uses shared gs recipe classes", () => {
  const source = fs.readFileSync(vanillaCounterPath, "utf8");
  assert.match(source, /gs-card/);
  assert.match(source, /gs-stack/);
  assert.match(source, /gs-button gs-button-primary/);
  assert.match(source, /gs-button gs-button-secondary/);
  assert.doesNotMatch(source, /\bbtn\b/);
  assert.doesNotMatch(source, /base-300|base-100/);
});

function findFiles(rootDir, ext) {
  const matches = [];

  for (const entry of fs.readdirSync(rootDir, { withFileTypes: true })) {
    const fullPath = path.join(rootDir, entry.name);
    if (entry.isDirectory()) {
      matches.push(...findFiles(fullPath, ext));
      continue;
    }
    if (fullPath.endsWith(ext)) {
      matches.push(fullPath);
    }
  }

  return matches;
}
