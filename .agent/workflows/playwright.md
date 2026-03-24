---
description: Run Playwright E2E tests and view the app in browser
---

# Playwright Testing Workflow

## Running Tests

// turbo
1. List all available tests:
```bash
cd d:\coding\project-tachyon\frontend && npx playwright test --list
```

// turbo
2. Run all tests headlessly:
```bash
cd d:\coding\project-tachyon\frontend && npx playwright test
```

// turbo
3. Run a specific test file:
```bash
cd d:\coding\project-tachyon\frontend && npx playwright test <test-file-name>
```

// turbo
4. Run tests with UI mode for debugging:
```bash
cd d:\coding\project-tachyon\frontend && npx playwright test --ui
```

// turbo
5. View the HTML test report:
```bash
cd d:\coding\project-tachyon\frontend && npx playwright show-report
```

## Viewing the App

// turbo
6. Start the dev server (if not already running):
```bash
cd d:\coding\project-tachyon\frontend && npm run dev
```

7. Use the `browser_subagent` tool to navigate to `http://localhost:5173` and interact with the app.

## Test Files Location

E2E tests are located in: `d:\coding\project-tachyon\frontend\e2e\`

- `app.spec.ts` - Core app tests
- `dashboard.spec.ts` - Dashboard navigation tests
- `download_flow.spec.ts` - Download flow tests
- `settings.spec.ts` - Settings configuration tests
- `tools.spec.ts` - Speed test and analytics tests
- `mocks.ts` - Mock data and route handlers
