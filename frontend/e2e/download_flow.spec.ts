import { test, expect } from '@playwright/test';
import { mockWailsApi } from './mocks';

test.beforeEach(async ({ page }) => {
    page.on('console', msg => console.log(`BROWSER LOG: ${msg.text()}`));
    await mockWailsApi(page);
    await page.goto('/');
});

test.describe('Download Flow', () => {
    test('should add a valid download', async ({ page }) => {
        await page.locator('button').filter({ hasText: 'Add Download' }).click();
        await expect(page.getByText('Add New Download')).toBeVisible();

        const input = page.getByPlaceholder('https://example.com/file.zip');
        await input.fill('https://example.com/testfile.zip');
        await page.getByRole('button', { name: 'Next' }).click();

        await expect(page.getByText('testfile.zip')).toBeVisible(); // From mock ProbeURL
        await page.getByRole('button', { name: 'Start Download' }).click();

        // Should close modal and show success toast or similar
        await expect(page.getByText('Add New Download')).not.toBeVisible();
    });

    test('should handle probe failure', async ({ page }) => {
        await page.locator('button').filter({ hasText: 'Add Download' }).click();

        const input = page.getByPlaceholder('https://example.com/file.zip');
        await input.fill('https://example.com/fail.zip'); // Trigger mock error
        await page.getByRole('button', { name: 'Next' }).click();

        await expect(page.getByText('Probe failed')).toBeVisible();
    });

    test('should detect collision', async ({ page }) => {
        await page.locator('button').filter({ hasText: 'Add Download' }).click();

        const input = page.getByPlaceholder('https://example.com/file.zip');
        await input.fill('https://example.com/collision.zip'); // Trigger collision mock
        await page.getByRole('button', { name: 'Next' }).click();

        await expect(page.getByText('File already exists')).toBeVisible();
    });
});

test.describe('Download Lifecycle (Context Menu)', () => {
    // Note: We need to mock an active download in Mocks or add one here
    // Since GetTasks returns [], we simulate adding one via event or relying on previous state if not isolated
    // But mocks are fresh every test. Let's rely on AddDownload flow or mock GetTasks to return one.
    // For now we will rely on updating the mock dynamically if possible, or just skip if complex.
    // BETTER: Update the mock for this specific test file or test case.

    // Playwright allows overriding mocks per test if we expose a helper, but our mockWailsApi is global.
    // We can use page.addInitScript again to override GetTasks for this test.

    test('should show context menu actions', async ({ page }) => {
        // Create a download first (E2E style)
        await page.locator('button').filter({ hasText: 'Add Download' }).click();
        await page.getByPlaceholder('https://example.com/file.zip').fill('https://example.com/menu_test.zip');
        await page.getByRole('button', { name: 'Next' }).click();
        await expect(page.getByText('menu_test.zip')).toBeVisible();
        await page.getByRole('button', { name: 'Start Download' }).click();

        // Download should appear in list
        // Note: AddDownload mock returns "task-id-1".
        // GetTasks mock returns []. 
        // Component state 'downloads' relies on AddDownload returning success AND THEN fetching or receiving event?
        // useTachyon: onAdd calls App.AddDownload. Then EXPECTS event 'download:added' or refreshes?
        // useTachyon doesn't auto-refresh list on Add, it listens to events.
        // We need to Mock the 'download:added' event or Mock GetTasks to include it.

        // Let's emit the event manually from browser side since we exposed runtime
        await page.evaluate(() => {
            // @ts-ignore
            window.go.runtime.EventsEmit('download:added', {
                id: "menu-test-id",
                url: "https://example.com/menu_test.zip",
                filename: "menu_test.zip",
                size: 1024,
                status: "downloading",
                progress: 0
            });
        });

        const row = page.getByText('menu_test.zip');
        await expect(row).toBeVisible();

        await row.click({ button: 'right' });
        await expect(page.getByText('Pause')).toBeVisible();
        await expect(page.getByText('Copy Link')).toBeVisible();
    });
});
