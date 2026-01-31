import { test, expect } from '@playwright/test';
import { mockWailsApi } from './mocks';

test.beforeEach(async ({ page }) => {
    await mockWailsApi(page);
    await page.goto('/');
});

test.describe('Settings Configuration', () => {
    test.beforeEach(async ({ page }) => {
        await page.getByRole('button', { name: 'Settings' }).click();
    });

    test('general tab should show danger zone', async ({ page }) => {
        await expect(page.getByText('Danger Zone')).toBeVisible();
        await expect(page.getByRole('button', { name: 'Reset Everything' })).toBeVisible();
    });

    test('mcp tab interactions', async ({ page }) => {
        await page.getByText('MCP Server').click(); // Tab
        await expect(page.getByText('Model Context Protocol')).toBeVisible();

        // Toggle MCP
        const toggle = page.locator('input[type="checkbox"]');
        await toggle.click();
        // Since mock GetEnableAI returns false, clicking should enable it locally (optimistic UI) or trigger SetEnableAI
    });

    test('security tab shows audit logs', async ({ page }) => {
        await page.getByText('Security').click(); // Tab
        await expect(page.getByText('Security Dashboard')).toBeVisible();

        // Check for mocked log entry
        await expect(page.getByText('User logged in')).toBeVisible();
        await expect(page.getByText('LOGIN')).toBeVisible();
    });
});
