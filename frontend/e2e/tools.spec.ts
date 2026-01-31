import { test, expect } from '@playwright/test';
import { mockWailsApi } from './mocks';

test.beforeEach(async ({ page }) => {
    page.on('console', msg => console.log(`BROWSER LOG: ${msg.text()}`));
    await mockWailsApi(page);
    await page.goto('/');
});

test.describe('Speed Test Tool', () => {
    test('should run speed test', async ({ page }) => {
        await page.getByRole('button', { name: 'Speed Test' }).click();
        await expect(page.getByText('Network Speed Test')).toBeVisible();

        const runBtn = page.getByRole('button', { name: 'Start' });
        await expect(runBtn).toBeVisible();
        await runBtn.click();

        // Specific result from mock
        await expect(page.getByText('15.5')).toBeVisible(); // Download MB/s
        await expect(page.getByText('5.2')).toBeVisible();  // Upload MB/s
        await expect(page.getByText('25 ms')).toBeVisible();  // Latency
    });
});

test.describe('Analytics Tool', () => {
    test('should render analytics charts', async ({ page }) => {
        await page.getByRole('button', { name: 'Analytics' }).click();

        // Key sections
        await expect(page.getByText('Network Activity')).toBeVisible();
        await expect(page.getByText('Library Composition')).toBeVisible();

        // Stats
        await expect(page.getByText('Lifetime Download')).toBeVisible();
        await expect(page.getByText('Disk Used')).toBeVisible();
    });
});
