import { Page } from '@playwright/test';

export const mockWailsApi = async (page: Page) => {
    await page.addInitScript(() => {
        // @ts-ignore
        window.go = {
            app: {
                App: {
                    GetDownloadLocations: async () => [],
                    GetTasks: async () => [], // Return empty task list
                    GetAnalytics: async () => ({ daily_history: {}, disk_usage: { free_gb: 100, percent: 10 } }),
                    GetLifetimeStats: async () => ({ total_bytes: 0 }),
                    ProbeURL: async (url: string) => ({
                        status: 200,
                        filename: "testfile.zip",
                        size: 1024 * 1024 * 50 // 50MB
                    }),
                    CheckHistory: async () => false,
                    CheckCollision: async () => ({ exists: false }),
                    AddDownload: async () => "task-id-1",
                    AddDownloadWithFilename: async () => "task-id-1",
                    AddDownloadWithOptions: async () => "task-id-1",
                    AddDownloadWithParams: async () => "task-id-1",
                    PauseDownload: async () => { },
                    ResumeDownload: async () => { },
                    StopDownload: async () => { },
                    DeleteDownload: async () => { },
                    OpenFile: async () => { },
                    OpenFolder: async () => { },
                    ReorderDownload: async () => { },
                    SetPriority: async () => { },
                    PauseAllDownloads: async () => { },
                    ResumeAllDownloads: async () => { },
                    GetNetworkHealth: async () => ({ level: 'normal' }),
                    GetEnableAI: async () => false,
                    GetAIPort: async () => 4444,
                    GetAIMaxConcurrent: async () => 5,
                    GetAIToken: async () => "mock-token",
                    GetRecentAuditLogs: async () => [],
                    FactoryReset: async () => { }, // Mock Reset
                    SetMaxConcurrentDownloads: async () => { },
                    SetEnableAI: async () => { },
                    SetAIPort: async () => { },
                    SetAIMaxConcurrent: async () => { },
                    SetGlobalSpeedLimit: async () => { },
                    UpdateSettings: async () => { },
                }
            },
            runtime: {
                // @ts-ignore
                EventsOn: (e, c) => { },
                // @ts-ignore
                EventsOff: (e) => { },
                // @ts-ignore
                EventsOnMultiple: (e, c, m) => { },
            }
        };
        // @ts-ignore
        window.runtime = window.go.runtime;
    });
};
