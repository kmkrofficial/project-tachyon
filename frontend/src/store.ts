import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface SettingsState {
    maxConcurrentDownloads: number;
    threadsPerDownload: number;
    globalSpeedLimit: number; // Bytes per second, 0 = unlimited
    downloadPath: string;
    theme: 'dark' | 'light' | 'black' | 'system';
    sidebarCollapsed: boolean;
    fileCategorization: boolean;
    downloadRetries: number;
    timeFormat: 'relative' | 'absolute';
    startOnBoot: boolean;
    closeToTray: boolean;
    sizeUnit: 'auto' | 'KB' | 'MB' | 'GB';
    quickDownload: boolean;
    completedClickAction: 'open-file' | 'open-folder';
    schedulerTime: string; // HH:MM format for global scheduler

    setMaxConcurrentDownloads: (n: number) => void;
    setThreadsPerDownload: (n: number) => void;
    setGlobalSpeedLimit: (n: number) => void;
    setDownloadPath: (path: string) => void;
    setTheme: (theme: 'dark' | 'light' | 'black' | 'system') => void;
    setSidebarCollapsed: (collapsed: boolean) => void;
    setFileCategorization: (enabled: boolean) => void;
    setDownloadRetries: (n: number) => void;
    setTimeFormat: (format: 'relative' | 'absolute') => void;
    setStartOnBoot: (enabled: boolean) => void;
    setCloseToTray: (enabled: boolean) => void;
    setSizeUnit: (unit: 'auto' | 'KB' | 'MB' | 'GB') => void;
    setQuickDownload: (enabled: boolean) => void;
    setCompletedClickAction: (action: 'open-file' | 'open-folder') => void;
    setSchedulerTime: (time: string) => void;
}

export const useSettingsStore = create<SettingsState>()(
    persist(
        (set) => ({
            maxConcurrentDownloads: 3,
            threadsPerDownload: 16,
            globalSpeedLimit: 0,
            downloadPath: '',
            theme: 'dark',
            sidebarCollapsed: false,
            fileCategorization: true,
            downloadRetries: 3,
            timeFormat: 'relative',
            startOnBoot: false,
            closeToTray: true,
            sizeUnit: 'auto',
            quickDownload: false,
            completedClickAction: 'open-file',
            schedulerTime: '02:00',

            setMaxConcurrentDownloads: (n) => set({ maxConcurrentDownloads: n }),
            setThreadsPerDownload: (n) => set({ threadsPerDownload: n }),
            setGlobalSpeedLimit: (n) => set({ globalSpeedLimit: n }),
            setDownloadPath: (path) => set({ downloadPath: path }),
            setTheme: (theme) => set({ theme }),
            setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
            setFileCategorization: (enabled) => set({ fileCategorization: enabled }),
            setDownloadRetries: (n) => set({ downloadRetries: n }),
            setTimeFormat: (format) => set({ timeFormat: format }),
            setStartOnBoot: (enabled) => set({ startOnBoot: enabled }),
            setCloseToTray: (enabled) => set({ closeToTray: enabled }),
            setSizeUnit: (unit) => set({ sizeUnit: unit }),
            setQuickDownload: (enabled) => set({ quickDownload: enabled }),
            setCompletedClickAction: (action) => set({ completedClickAction: action }),
            setSchedulerTime: (time) => set({ schedulerTime: time }),
        }),
        {
            name: 'tachyon-settings',
        }
    )
);
