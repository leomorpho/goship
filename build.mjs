import esbuild from "esbuild";
import sveltePlugin from "esbuild-svelte";
import fs from "fs/promises"; // Use the fs/promises API for async/await
import path from "path";
import sveltePreprocess from "svelte-preprocess";

const svelteEntrypointsDir = "javascript/svelte";
const outputDir = "static";

// Define an asynchronous function to handle the build process
async function build() {
  try {
    // Read all entry point files in the entrypoints directory
    const files = await fs.readdir(svelteEntrypointsDir);
    // Filter for .js files only
    const entryPoints = files
      .filter((file) => path.extname(file) === ".js")
      .map((file) => path.join(svelteEntrypointsDir, file));

    console.log("Svelte entry points:", entryPoints);
    // Bundle all Svelte components into a single file
    const svelteResult = await esbuild.build({
      entryPoints: entryPoints,
      mainFields: ["svelte", "browser", "module", "main"],
      conditions: ["svelte", "browser"],
      bundle: true,
      outfile: path.join(outputDir, "svelte_bundle.js"),
      minify: true,
      sourcemap: true,
      format: "esm",
      plugins: [
        sveltePlugin({
          preprocess: sveltePreprocess({ typescript: true }),
        }),
      ],
      metafile: true,
    });

    // Write the metafile for Svelte bundle
    await fs.writeFile(
      path.join(outputDir, "meta_svelte_bundle.json"),
      JSON.stringify(svelteResult.metafile)
    );

    // Bundle vanilla JS or other assets as needed
    const vanillaResult = await esbuild.build({
      entryPoints: ["javascript/vanilla/main.js"], // Entry point for your vanilla JS
      bundle: true,
      outfile: "static/vanilla_bundle.js", // Output file for vanilla JS
      minify: true,
      sourcemap: true,
      metafile: true,
    });

    // Write the metafile to disk and open with https://esbuild.github.io/analyze/
    await fs.writeFile(
      path.join(outputDir, "meta_vanilla_bundle.json"),
      JSON.stringify(vanillaResult.metafile)
    );

    // NOTE: the below pattern can be used to create standalone island components that can be loaded only when needed in the FE
    // await esbuild.build({
    //   entryPoints: ["javascript/vanilla/audio_player.js"], // Entry point for your vanilla JS
    //   bundle: true,
    //   outfile: "static/audio_player.js", // Output file for vanilla JS
    //   minify: true,
    //   sourcemap: true,
    //   metafile: true,
    // });

    console.log("Build completed successfully");
  } catch (error) {
    console.error("Build failed:", error);
    process.exit(1);
  }
}

// Execute the build function
build();
