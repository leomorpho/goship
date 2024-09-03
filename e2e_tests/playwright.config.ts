import { PlaywrightTestConfig } from "@playwright/test";

const config: PlaywrightTestConfig = {
  projects: [
    {
      name: "Chrome",
      use: { browserName: "chromium" },
    },
  ],
  workers: 1, // Adjust the number of workers as needed based on your CI environment capabilities
  retries: 2,
};
export default config;
