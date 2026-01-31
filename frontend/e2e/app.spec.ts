import { test, expect } from '@playwright/test';
import { mockWailsApi } from './mocks';

test.beforeEach(async ({ page }) => {
    page.on('console', msg => console.log(`BROWSER LOG: ${msg.text()}`));
    await mockWailsApi(page);
    await page.goto('/');
});

test.describe('Dashboard', () => {
    test('should load dashboard with correct title', async ({ page }) => {
        await expect(page).toHaveTitle(/Tachyon/i);
        await expect(page.getByText('Dashboard')).toBeVisible();
        await expect(page.getByText('Overview')).toBeVisible();
    });

    test('should show network health indicator', async ({ page }) => {
        // Mock returned 'normal', so it should say 'Healthy'
        await expect(page.getByTitle(/Network: Healthy/i)).toBeVisible();
    });
});

test.describe('Add Download Flow', () => {
    test('should open modal and probe URL', async ({ page }) => {
        await page.locator('button').filter({ hasText: 'Add Download' }).click();
        await expect(page.getByText('Add New Download')).toBeVisible();

        const input = page.getByPlaceholder('https://example.com/file.zip');
        await input.fill('https://example.com/testfile.zip');

        // Click Next (which triggers ProbeURL mock)
        await page.getByRole('button', { name: 'Next' }).click();

        // Mock returns "testfile.zip", so check if it appears
        await expect(page.getByText('testfile.zip')).toBeVisible();

        // Check if Start Download button appears
        await expect(page.getByRole('button', { name: 'Start Download' })).toBeVisible();
    });
});

test.describe('Settings & Factory Reset', () => {
    test('should open settings modal', async ({ page }) => {
        // Click Settings icon in sidebar (it might rely on aria-label or just text if available, sidebar usually has icons)
        // Sidebar items: Dashboard, Speedtest, Analytics, Settings
        await page.locator('button').filter({ hasText: 'Settings' }).click();
        await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible();
    });

    test('should navigate to danger zone and trigger factory reset', async ({ page }) => {
        await page.locator('button').filter({ hasText: 'Settings' }).click();

        // Danger Zone is in General tab (default)
        await expect(page.getByText('Danger Zone')).toBeVisible();

        // Click Reset Everything
        await page.getByRole('button', { name: 'Reset Everything' }).click();

        // Expect confirmation
        await expect(page.getByText('Are you sure?')).toBeVisible();

        // Click Yes, Wipe It
        // We mocked FactoryReset, so this should not actually wipe anything but should succeed
        await page.getByRole('button', { name: 'Yes, Wipe It' }).click();
    });
});
