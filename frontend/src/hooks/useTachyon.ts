import { useEffect, useState } from "react";
// @ts-ignore
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
// @ts-ignore
import * as App from "../../wailsjs/go/main/App";

import { DownloadItem } from "../types";

export function useTachyon() {
    const [downloads, setDownloads] = useState<Record<string, DownloadItem>>({});

    useEffect(() => {
        // Load History on Mount
        const loadHistory = async () => {
            if (App && App.GetTasks) {
                try {
                    const tasks = await App.GetTasks();
                    // Convert Array to Record
                    const history: Record<string, DownloadItem> = {};
                    tasks.forEach((t: any) => {
                        history[t.id] = {
                            id: t.id,
                            url: t.url,
                            filename: t.filename,
                            progress: t.progress,
                            size: t.size,
                            status: t.status as any,
                            path: t.path,
                            created_at: t.created_at,
                            // Derived/Default values
                            speed_MBs: 0,
                            eta: "--"
                        };
                    });
                    // Merge with existing (though on mount existing is empty)
                    setDownloads(prev => ({ ...prev, ...history }));
                } catch (e) {
                    console.error("Failed to load history", e);
                }
            }
        };
        loadHistory();

        // Listen for Progress (existing code...)
        const cleanupProgress = EventsOn("download:progress", (data: any) => {
            setDownloads((prev) => ({
                ...prev,
                [data.id]: {
                    ...prev[data.id],
                    id: data.id,
                    filename: data.filename || prev[data.id]?.filename || "Unknown",
                    progress: data.progress,
                    speed_MBs: data.speed_MBs,
                    eta: data.eta,
                    status: "downloading",
                },
            }));
        });

        // Listen for Completion
        const cleanupCompleted = EventsOn("download:completed", (data: any) => {
            setDownloads((prev) => ({
                ...prev,
                [data.id]: {
                    ...prev[data.id],
                    status: "completed",
                    progress: 100,
                    path: data.path
                }
            }))
        });

        // Listen for Errors
        const cleanupFailed = EventsOn("download:failed", (data: any) => {
            setDownloads((prev) => ({
                ...prev,
                [data.id]: {
                    ...prev[data.id],
                    status: "error",
                    error: data.error
                }
            }))
        });

        return () => {
            // events cleanup
        };
    }, []);

    const addDownload = async (url: string) => {
        if (App && App.AddDownload) {
            try {
                const id = await App.AddDownload(url);
                // The progress event / GetTasks will eventually populate it, 
                // but setting initial state allows instant UI feedback
                setDownloads(prev => ({
                    ...prev,
                    [id]: {
                        id,
                        url: url,
                        filename: "Initializing...",
                        progress: 0,
                        size: 0,
                        speed_MBs: 0,
                        eta: "--",
                        status: "downloading",
                        created_at: new Date().toISOString()
                    }
                }))
                return id;
            } catch (e: any) {
                console.error("Failed to add download", e);
                throw e;
            }
        } else {
            throw new Error("Backend not connected");
        }
    };

    const openFolder = async (id: string) => {
        if (App && App.OpenFolder) {
            await App.OpenFolder(id);
        }
    };

    const openFile = async (id: string) => {
        if (App && App.OpenFile) {
            await App.OpenFile(id);
        }
    };

    const runSpeedTest = async () => {
        if (App && App.RunNetworkSpeedTest) {
            return await App.RunNetworkSpeedTest();
        }
        return null;
    }

    const getLifetimeStats = async () => {
        if (App && App.GetLifetimeStats) {
            return await App.GetLifetimeStats();
        }
        return 0;
    }

    return {
        downloads,
        addDownload,
        openFolder,
        openFile,
        runSpeedTest,
        getLifetimeStats
    };
}
