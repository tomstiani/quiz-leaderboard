import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:18080',
    trace: 'retain-on-failure',
  },
  webServer: [
    {
      command: "cd .. && node -e \"require('fs').rmSync('.e2e-data',{recursive:true,force:true})\" && CONFIG_FILE=config.e2e.json go run .",
      url: 'http://127.0.0.1:18080/api/health',
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: 'node e2e/fake-vision.mjs',
      url: 'http://127.0.0.1:18081',
      reuseExistingServer: false,
    },
  ],
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
})
