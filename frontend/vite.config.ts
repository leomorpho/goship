import fs from "node:fs";
import path from "node:path";
import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import sveltePreprocess from "svelte-preprocess";

const frontendRoot = __dirname;
const outputDir = path.resolve(frontendRoot, "../app/static");
const islandsDir = path.resolve(frontendRoot, "islands");
const islandsRuntimeEntry = path.resolve(frontendRoot, "javascript/vanilla/islands-runtime.js");
const vanillaEntry = path.resolve(frontendRoot, "javascript/vanilla/main.js");
const staticURLPrefix = "/files";

function collectIslandEntries(rootDir: string): Record<string, string> {
  const entries: Record<string, string> = {};

  if (!fs.existsSync(rootDir)) {
    return entries;
  }

  const walk = (dir: string) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const fullPath = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(fullPath);
        continue;
      }

      if (!/\.(svelte|tsx|jsx|vue)$/.test(entry.name)) {
        continue;
      }

      const relPath = path.relative(rootDir, fullPath);
      const normalized = relPath.split(path.sep).join("/");
      const entryName = normalized.replace(/\.(svelte|tsx|jsx|vue)$/, "");
      entries[`islands/${entryName}`] = fullPath;
    }
  };

  walk(rootDir);
  return entries;
}

function islandNameFromFacadeModuleId(id: string | null | undefined): string {
  if (!id) {
    return "";
  }

  const relPath = path.relative(islandsDir, id);
  const normalized = relPath.split(path.sep).join("/");
  return normalized.replace(/\.(svelte|tsx|jsx|vue)$/, "");
}

function islandsManifestPlugin() {
  return {
    name: "goship-islands-manifest",
    generateBundle(_: unknown, bundle: Record<string, any>) {
      const manifest: Record<string, string> = {};

      for (const [fileName, artifact] of Object.entries(bundle)) {
        if (artifact.type !== "chunk" || !artifact.isEntry) {
          continue;
        }
        if (!artifact.name.startsWith("islands/")) {
          continue;
        }

        const islandName = islandNameFromFacadeModuleId(artifact.facadeModuleId);
        if (!islandName) {
          continue;
        }

        manifest[islandName] = `${staticURLPrefix}/${fileName}`;
      }

      this.emitFile({
        type: "asset",
        fileName: "islands-manifest.json",
        source: JSON.stringify(manifest, null, 2) + "\n",
      });
    },
  };
}

export default defineConfig({
  plugins: [
    svelte({
      preprocess: sveltePreprocess({ typescript: true }),
    }),
    islandsManifestPlugin(),
  ],
  publicDir: false,
  build: {
    outDir: outputDir,
    emptyOutDir: false,
    sourcemap: true,
    manifest: false,
    rollupOptions: {
      input: {
        islands_runtime: islandsRuntimeEntry,
        vanilla_bundle: vanillaEntry,
        ...collectIslandEntries(islandsDir),
      },
      output: {
        entryFileNames(chunkInfo) {
          if (chunkInfo.name === "islands_runtime") {
            return "islands-runtime.js";
          }
          if (chunkInfo.name === "vanilla_bundle") {
            return "vanilla_bundle.js";
          }
          return "[name]-[hash].js";
        },
        chunkFileNames: "islands/chunks/[name]-[hash].js",
        assetFileNames(assetInfo) {
          if (assetInfo.name === "islands-manifest.json") {
            return "islands-manifest.json";
          }
          return "islands/assets/[name]-[hash][extname]";
        },
      },
    },
  },
});
