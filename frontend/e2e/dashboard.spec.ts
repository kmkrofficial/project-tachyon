import { test, expect } from '@playwright/test';
import { mockWailsApi } from './mocks';

test.beforeEach(async ({ page }) => {
    page.on('console', msg => console.log(`BROWSER LOG: ${msg.text()}`));
    await mockWailsApi(page);
    await page.goto('/');
});

test.describe('Dashboard Navigation & Layout', () => {
    test('default route should cover Dashboard elements', async ({ page }) => {
        await expect(page).toHaveTitle(/Tachyon/i);
        // Sidebar active state
        const dashboardBtn = page.getByRole('button', { name: 'Dashboard' });
        await expect(dashboardBtn).toHaveClass(/bg-slate-800/);

        // Header info
        await expect(page.getByText('Overview')).toBeVisible();
    });

    test('navigation tabs switch content', async ({ page }) => {
        // Switch to Analytics
        await page.getByRole('button', { name: 'Analytics' }).click();
        await expect(page.getByText('Download Activity')).toBeVisible(); // Chart title or similar

        // Switch to Settings
        await page.getByRole('button', { name: 'Settings' }).click();
        await expect(page.getByText('General')).toBeVisible();
    });
});

test.describe('Header Global Actions', () => {
    test('pause/resume buttons are interactive', async ({ page }) => {
        const pauseBtn = page.locator('button[title="Pause All"]');
        await expect(pauseBtn).toBeVisible();
        await pauseBtn.click();
        // Verification relies on mock call spy (advanced) or just no crash for now

        const resumeBtn = page.locator('button[title="Resume All"]');
        await expect(resumeBtn).toBeVisible();
        await resumeBtn.click();
    });

    test('network health indicator', async ({ page }) => {
        const indicator = page.locator('div[title*="Network:"]');
        await expect(indicator).toBeVisible();
        await expect(indicator).toContainText('Healthy'); // Mock returns 'normal'
    });
});
