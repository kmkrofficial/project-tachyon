import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useTheme } from '../../hooks/useTheme';
import { useSettingsStore } from '../../store';

describe('useTheme', () => {
    beforeEach(() => {
        document.documentElement.classList.remove('dark');
        // Reset store to default
        useSettingsStore.setState({ theme: 'dark' });
    });

    it('adds dark class when theme is dark', () => {
        useSettingsStore.setState({ theme: 'dark' });
        renderHook(() => useTheme());
        expect(document.documentElement.classList.contains('dark')).toBe(true);
    });

    it('removes dark class when theme is light', () => {
        document.documentElement.classList.add('dark');
        useSettingsStore.setState({ theme: 'light' });
        renderHook(() => useTheme());
        expect(document.documentElement.classList.contains('dark')).toBe(false);
    });

    it('follows system preference when theme is system', () => {
        // Assign matchMedia directly since jsdom doesn't define it
        window.matchMedia = vi.fn().mockReturnValue({
            matches: true,
            media: '(prefers-color-scheme: dark)',
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
        });

        useSettingsStore.setState({ theme: 'system' });
        renderHook(() => useTheme());
        expect(document.documentElement.classList.contains('dark')).toBe(true);
    });

    it('follows system light preference', () => {
        window.matchMedia = vi.fn().mockReturnValue({
            matches: false,
            media: '(prefers-color-scheme: dark)',
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
        });

        useSettingsStore.setState({ theme: 'system' });
        renderHook(() => useTheme());
        expect(document.documentElement.classList.contains('dark')).toBe(false);
    });

    it('returns current theme and setter', () => {
        useSettingsStore.setState({ theme: 'dark' });
        const { result } = renderHook(() => useTheme());
        expect(result.current.theme).toBe('dark');
        expect(typeof result.current.setTheme).toBe('function');
    });
});
