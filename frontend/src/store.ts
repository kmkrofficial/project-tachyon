import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface SettingsState {
    maxConcurrentDownloads: number;
    threadsPerDownload: number;
    globalSpeedLimit: number; // Bytes per second, 0 = unlimited
    downloadPath: string;
    theme: 'dark' | 'light' | 'system';

    setMaxConcurrentDownloads: (n: number) => void;
    setThreadsPerDownload: (n: number) => void;
    setGlobalSpeedLimit: (n: number) => void;
    setDownloadPath: (path: string) => void;
    setTheme: (theme: 'dark' | 'light' | 'system') => void;
}

export const useSettingsStore = create<SettingsState>()(
    persist(
        (set) => ({
            maxConcurrentDownloads: 3,
            threadsPerDownload: 16,
            globalSpeedLimit: 0,
            downloadPath: '', // Empty means default
            theme: 'dark',

            setMaxConcurrentDownloads: (n) => set({ maxConcurrentDownloads: n }),
            setThreadsPerDownload: (n) => set({ threadsPerDownload: n }),
            setGlobalSpeedLimit: (n) => set({ globalSpeedLimit: n }),
            setDownloadPath: (path) => set({ downloadPath: path }),
            setTheme: (theme) => set({ theme }),
        }),
        {
            name: 'tachyon-settings',
        }
    )
);
