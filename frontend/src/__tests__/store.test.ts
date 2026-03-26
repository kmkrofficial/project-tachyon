import { describe, it, expect } from 'vitest';
import { useSettingsStore } from '../store';
import { act } from '@testing-library/react';

describe('useSettingsStore', () => {
    it('has correct default values', () => {
        const state = useSettingsStore.getState();
        expect(state.maxConcurrentDownloads).toBe(3);
        expect(state.threadsPerDownload).toBe(16);
        expect(state.globalSpeedLimit).toBe(0);
        expect(state.downloadPath).toBe('');
        expect(state.theme).toBe('dark');
        expect(state.sidebarCollapsed).toBe(false);
    });

    it('sets maxConcurrentDownloads', () => {
        act(() => useSettingsStore.getState().setMaxConcurrentDownloads(8));
        expect(useSettingsStore.getState().maxConcurrentDownloads).toBe(8);
    });

    it('sets threadsPerDownload', () => {
        act(() => useSettingsStore.getState().setThreadsPerDownload(4));
        expect(useSettingsStore.getState().threadsPerDownload).toBe(4);
    });

    it('sets globalSpeedLimit', () => {
        act(() => useSettingsStore.getState().setGlobalSpeedLimit(1024));
        expect(useSettingsStore.getState().globalSpeedLimit).toBe(1024);
    });

    it('sets downloadPath', () => {
        act(() => useSettingsStore.getState().setDownloadPath('/tmp/downloads'));
        expect(useSettingsStore.getState().downloadPath).toBe('/tmp/downloads');
    });

    it('sets theme', () => {
        act(() => useSettingsStore.getState().setTheme('light'));
        expect(useSettingsStore.getState().theme).toBe('light');
        act(() => useSettingsStore.getState().setTheme('system'));
        expect(useSettingsStore.getState().theme).toBe('system');
    });

    it('sets sidebarCollapsed', () => {
        act(() => useSettingsStore.getState().setSidebarCollapsed(true));
        expect(useSettingsStore.getState().sidebarCollapsed).toBe(true);
        act(() => useSettingsStore.getState().setSidebarCollapsed(false));
        expect(useSettingsStore.getState().sidebarCollapsed).toBe(false);
    });
});
