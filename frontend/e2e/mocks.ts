import { Page } from '@playwright/test';

export const mockWailsApi = async (page: Page) => {
    await page.addInitScript(() => {
        const mockEvents: Record<string, Function[]> = {};

        // @ts-ignore
        window.go = {
            app: {
                App: {
                    // Downloads
                    GetTasks: async () => [],
                    GetDownloadLocations: async () => [{ name: "Default", path: "C:\\Downloads" }],
                    ProbeURL: async (url: string) => {
                        if (url.includes("fail")) throw new Error("Probe failed");
                        if (url.includes("404")) return { status: 404, size: 0, filename: "" };

                        let filename = "testfile.zip";
                        if (url.includes("collision")) filename = "collision.zip";
                        else if (url.includes("/")) {
                            const parts = url.split("/");
                            const last = parts[parts.length - 1];
                            if (last && last.includes(".")) filename = last;
                        }

                        return {
                            status: 200,
                            filename: filename,
                            size: 1024 * 1024 * 50 // 50MB
                        };
                    },
                    CheckHistory: async (url: string) => url.includes("history"),
                    CheckCollision: async (filename: string) => ({ exists: filename.includes("collision") }),
                    AddDownload: async () => "task-id-1",
                    AddDownloadWithFilename: async () => "task-id-1",
                    AddDownloadWithOptions: async () => "task-id-1",
                    AddDownloadWithParams: async () => "task-id-1",
                    PauseDownload: async (id: string) => { },
                    ResumeDownload: async (id: string) => { },
                    StopDownload: async (id: string) => { },
                    DeleteDownload: async (id: string, deleteFile: boolean) => { },
                    OpenFile: async (id: string) => { },
                    OpenFolder: async (id: string) => { },
                    ReorderDownload: async (id: string, dir: string) => { },
                    SetPriority: async (id: string, p: number) => { },
                    PauseAllDownloads: async () => { },
                    ResumeAllDownloads: async () => { },

                    // Analytics & Dashboard
                    GetAnalytics: async () => ({
                        daily_history: { [new Date().toISOString().split('T')[0]]: 1024 * 1024 * 100 },
                        disk_usage: { free_gb: 100, percent: 45 }
                    }),
                    GetLifetimeStats: async () => ({ total_bytes: 1024 * 1024 * 1024 * 5 }), // 5GB
                    GetNetworkHealth: async () => ({ level: 'normal' }),
                    RunNetworkSpeedTest: async () => ({
                        download_speed: 15.5,
                        upload_speed: 5.2,
                        latency: 25,
                        server: "Test Server"
                    }),

                    // Settings & MCP
                    GetEnableAI: async () => false,
                    SetEnableAI: async (v: boolean) => { },
                    GetAIPort: async () => 4444,
                    SetAIPort: async (p: number) => { },
                    GetAIMaxConcurrent: async () => 5,
                    SetAIMaxConcurrent: async (c: number) => { },
                    GetAIToken: async () => "mock-token-123",
                    SetMaxConcurrentDownloads: async (c: number) => { },
                    SetGlobalSpeedLimit: async (s: number) => { },
                    UpdateSettings: async (s: any) => { },
                    FactoryReset: async () => { },

                    // Security
                    GetRecentAuditLogs: async () => [
                        { id: "1", timestamp: new Date().toISOString(), source_ip: "127.0.0.1", action: "LOGIN", status: 200, details: "User logged in" }
                    ],
                }
            },
            runtime: {
                // @ts-ignore
                EventsOn: (event, callback) => {
                    if (!mockEvents[event]) mockEvents[event] = [];
                    mockEvents[event].push(callback);
                    // Return unsubscribe/cleanup
                    return () => {
                        mockEvents[event] = mockEvents[event].filter(cb => cb !== callback);
                    }
                },
                // @ts-ignore
                EventsOff: (event) => {
                    delete mockEvents[event];
                },
                // @ts-ignore
                EventsOnMultiple: (event, callback, max) => { },
                // @ts-ignore
                EventsEmit: (event, data) => {
                    if (mockEvents[event]) {
                        mockEvents[event].forEach(cb => cb(data));
                    }
                }
            }
        };
        // @ts-ignore
        window.runtime = window.go.runtime;
    });
};
