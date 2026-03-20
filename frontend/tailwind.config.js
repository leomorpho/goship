const path = require("node:path");

const repoRoot = path.resolve(__dirname, "..");

module.exports = {
  content: [
    path.join(__dirname, "javascript/**/*.{js,ts,jsx,tsx,svelte,vue}"),
    path.join(__dirname, "islands/**/*.{js,ts,jsx,tsx,svelte,vue}"),
    path.join(repoRoot, "app/views/**/*.templ"),
    path.join(__dirname, "node_modules/flowbite/**/*.js"),
  ],
  theme: {
    extend: {
      backdropBlur: {
        xs: "2px",
      },
      fontFamily: {
        PlayfairDisplay: ["Playfair Display", "serif"],
      },
    },
  },
  // https://themes.ionevolve.com/
  daisyui: {
    themes: [
      {
        lightmode: {
          // Change to any existing daisyui theme or make your own
          ...require("daisyui/src/theming/themes")["cmyk"],
          // Edit styles if required
          primary: "white",
          secondary: "#DEFBFB",
          accent: "#FA6A7D",
          neutral: "#919191",
          "base-100": "",
          info: "#623CEA",
          success: "#87FF65",
          warning: "#FFC759",
          error: "#A30000",
        },
      },
      {
        darkmode: {
          // Change to any existing daisyui theme or make your own
          ...require("daisyui/src/theming/themes")["business"],
          // Edit styles if required
          primary: "#111827",
          secondary: "#222833",
          accent: "#FA6A7D",
          neutral: "#494949",
          "base-100": "#010D14",
          info: "#623CEA",
          success: "#80D569",
          warning: "#FFC759",
          error: "#A30000",
        },
      },
    ],
  },
  plugins: [require("daisyui"), require("flowbite/plugin")],
};
