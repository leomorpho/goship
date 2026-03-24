import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, "..");

export default {
  darkMode: "class",
  content: [
    path.join(__dirname, "javascript/**/*.{js,ts,jsx,tsx,svelte,vue}"),
    path.join(__dirname, "islands/**/*.{js,ts,jsx,tsx,svelte,vue}"),
    path.join(repoRoot, "app/views/**/*.templ"),
    path.join(repoRoot, "framework/**/*.templ"),
    path.join(repoRoot, "modules/**/*.templ"),
    path.join(__dirname, "node_modules/flowbite/**/*.js"),
  ],
  safelist: [
    "gs-button",
    "gs-button-primary",
    "gs-button-secondary",
    "gs-field-error",
    "gs-field-success",
    "gs-page",
    "gs-panel",
    "gs-text",
    "gs-title",
  ],
  theme: {
    extend: {
      backdropBlur: {
        xs: "2px",
      },
      borderRadius: {
        "gs-control": "var(--gs-radius-control)",
        "gs-panel": "var(--gs-radius-panel)",
      },
      boxShadow: {
        "gs-panel": "var(--gs-shadow-panel)",
      },
      colors: {
        gs: {
          accent: "var(--gs-color-accent)",
          "accent-contrast": "var(--gs-color-accent-contrast)",
          "accent-strong": "var(--gs-color-accent-strong)",
          bg: "var(--gs-color-background)",
          border: "var(--gs-color-border)",
          danger: "var(--gs-color-danger)",
          success: "var(--gs-color-success)",
          surface: "var(--gs-color-surface)",
          "surface-muted": "var(--gs-color-surface-muted)",
          text: "var(--gs-color-text)",
          "text-muted": "var(--gs-color-text-muted)",
        },
      },
      fontFamily: {
        PlayfairDisplay: ["Playfair Display", "serif"],
      },
      spacing: {
        "gs-page": "var(--gs-space-page)",
      },
    },
  },
  plugins: [require("franken-ui").default(), require("flowbite/plugin")],
};
