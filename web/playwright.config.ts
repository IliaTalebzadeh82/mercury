import { defineConfig, devices } from "@playwright/test";

const databaseURL = process.env.MERCURY_TEST_DATABASE_URL;
if (!databaseURL) throw new Error("MERCURY_TEST_DATABASE_URL is required for Playwright");
const executablePath = process.env.MERCURY_PLAYWRIGHT_EXECUTABLE_PATH;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  retries: process.env.CI ? 1 : 0,
  reporter: "list",
  use: { baseURL: "http://127.0.0.1:13000", trace: "retain-on-failure" },
  projects: [{
    name: "chromium",
    use: {
      ...devices["Desktop Chrome"],
      launchOptions: executablePath ? { executablePath } : undefined,
    },
  }],
  webServer: [
    {
      command: "go run ./cmd/mercury",
      cwd: "..",
      env: { ...process.env, MERCURY_DATABASE_URL: databaseURL, MERCURY_HTTP_ADDRESS: "127.0.0.1:18080", MERCURY_DIAGNOSTIC_API_ENABLED: "true" },
      url: "http://127.0.0.1:18080/readyz",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
    {
      command: "pnpm dev --hostname 127.0.0.1 --port 13000",
      env: { ...process.env, MERCURY_API_URL: "http://127.0.0.1:18080" },
      url: "http://127.0.0.1:13000/advertisers",
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
    },
  ],
});
