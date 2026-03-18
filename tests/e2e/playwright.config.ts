import { defineConfig, devices } from "@playwright/test";

const port = process.env.E2E_PORT ?? "8000";
const baseURL = process.env.E2E_BASE_URL ?? `http://127.0.0.1:${port}`;

export default defineConfig({
  testDir: "./tests",
  fullyParallel: false,
  workers: process.env.CI ? 1 : undefined,
  retries: process.env.CI ? 2 : 0,
  use: {
    baseURL,
    trace: "retain-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: {
    command: "go run ./cmd/web",
    cwd: "../..",
    url: `${baseURL}/up`,
    timeout: 120_000,
    reuseExistingServer: !process.env.CI,
    env: {
      ...process.env,
      PAGODA_HTTP_PORT: port,
      PAGODA_PROCESSES_WEB: "true",
      PAGODA_PROCESSES_WORKER: "false",
      PAGODA_PROCESSES_SCHEDULER: "false",
      PAGODA_PROCESSES_COLOCATED: "false",
    },
  },
});
