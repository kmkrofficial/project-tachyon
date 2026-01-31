import { useEffect, useState, useCallback } from "react";
// @ts-ignore
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
// @ts-ignore
import * as App from "../../wailsjs/go/app/App";
import prettyBytes from 'pretty-bytes';

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
                            queue_order: t.queue_order || 0,
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

        // Listen for Progress
        const cleanupProgress = EventsOn("download:progress", (data: any) => {
            setDownloads((prev) => ({
                ...prev,
                [data.id]: {
                    ...prev[data.id],
                    id: data.id,
                    filename: data.filename || prev[data.id]?.filename || "Unknown",
                    progress: data.progress ?? prev[data.id]?.progress ?? 0,
                    speed_MBs: data.speed != null ? data.speed / (1024 * 1024) : (prev[data.id]?.speed_MBs ?? 0),
                    eta: data.eta ?? prev[data.id]?.eta ?? "--",
                    size: data.total || prev[data.id]?.size || 0,
                    path: data.path || prev[data.id]?.path,
                    status: data.status || prev[data.id]?.status || "downloading",
                    url: data.url || prev[data.id]?.url || "",
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
                    path: data.path,
                    speed_MBs: 0,
                    eta: "Done"
                }
            }))
        });

        // Listen for Paused
        const cleanupPaused = EventsOn("download:paused", (data: any) => {
            setDownloads((prev) => ({
                ...prev,
                [data.id]: {
                    ...prev[data.id],
                    status: "paused",
                    speed_MBs: 0,
                    eta: "--"
                }
            }))
        });

        // Listen for Deleted
        const cleanupDeleted = EventsOn("download:deleted", (data: any) => {
            setDownloads((prev) => {
                const { [data.id]: _, ...rest } = prev;
                return rest;
            });
        });

        // Listen for Errors
        const cleanupFailed = EventsOn("download:failed", (data: any) => {
            setDownloads((prev) => ({
                ...prev,
                [data.id]: {
                    ...prev[data.id],
                    status: "error",
                    error: data.error,
                    speed_MBs: 0,
                    eta: "--"
                }
            }))
        });

        // Listen for Stopped
        const cleanupStopped = EventsOn("download:stopped", (data: any) => {
            setDownloads((prev) => ({
                ...prev,
                [data.id]: {
                    ...prev[data.id],
                    status: "stopped",
                    speed_MBs: 0,
                    eta: "--"
                }
            }))
        });

        // Listen for Queue Reordered
        const cleanupReordered = EventsOn("queue:reordered", () => {
            loadHistory();
        });

        return () => {
            EventsOff("download:progress");
            EventsOff("download:completed");
            EventsOff("download:paused");
            EventsOff("download:deleted");
            EventsOff("download:failed");
            EventsOff("download:stopped");
            EventsOff("queue:reordered");
        };
    }, []);

    const addDownload = useCallback(async (url: string, filename?: string, size?: number, path?: string, options?: any) => {
        if (App) {
            try {
                let id = "";
                // Use the new versatile method if available, fallback otherwise (though we expect it to be there)
                if (options && App.AddDownloadWithParams) {
                    id = await App.AddDownloadWithParams(url, path || "", filename || "", options);
                } else if (App.AddDownloadWithOptions) {
                    id = await App.AddDownloadWithOptions(url, path || "", filename || "");

                    // Also save the location if it's a custom path
                    // Logic for saving location removed as it's handled elsewhere or pending implementation

                } else if (filename) {
                    // Fallback
                    id = await App.AddDownloadWithFilename(url, filename);
                } else {
                    id = await App.AddDownload(url);
                }

                setDownloads(prev => ({
                    ...prev,
                    [id]: {
                        id,
                        url: url,
                        filename: filename || "Initializing...",
                        progress: 0,
                        size: size || 0,
                        speed_MBs: 0,
                        eta: "--",
                        status: "downloading",
                        path: path,
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
    }, []);

    const openFolder = useCallback(async (id: string) => {
        if (App && App.OpenFolder) {
            await App.OpenFolder(id);
        }
    }, []);

    const openFile = useCallback(async (id: string) => {
        if (App && App.OpenFile) {
            await App.OpenFile(id);
        }
    }, []);

    const runSpeedTest = useCallback(async () => {
        if (App && App.RunNetworkSpeedTest) {
            return await App.RunNetworkSpeedTest();
        }
        return null;
    }, []);

    const getLifetimeStats = useCallback(async () => {
        if (App && App.GetLifetimeStats) {
            return await App.GetLifetimeStats();
        }
        return 0;
    }, []);

    const [dailyData, setDailyData] = useState<string>("0 GB");
    const [analyticsData, setAnalyticsData] = useState<any>(null); // Full raw data
    const [diskUsage, setDiskUsage] = useState<any>({ free_gb: 0, percent: 0 });
    const [totalSpeed, setTotalSpeed] = useState<number>(0);

    useEffect(() => {
        // ... (existing history loading)

        // Poll for Dashboard Stats (every 2s)
        const interval = setInterval(async () => {
            if (App && App.GetAnalytics) {
                try {
                    const analytics = await App.GetAnalytics();
                    setAnalyticsData(analytics);

                    // Find today's bytes
                    const today = new Date().toISOString().split('T')[0];
                    if (analytics.daily_history && analytics.daily_history[today]) {
                        setDailyData(prettyBytes(analytics.daily_history[today]));
                    } else {
                        setDailyData("0 B");
                    }

                    setDiskUsage(analytics.disk_usage);

                } catch (e) {
                    // console.error(e);
                }
            }
        }, 10000); // Poll every 10 seconds

        return () => clearInterval(interval);
    }, []);

    const reorderDownload = useCallback(async (id: string, direction: string) => {
        if (App && App.ReorderDownload) {
            await App.ReorderDownload(id, direction);
            // The queue:reordered event will trigger reload
        }
    }, []);

    const setPriority = useCallback(async (id: string, priority: number) => {
        if (App && App.SetPriority) {
            await App.SetPriority(id, priority);
        }
    }, []);

    // Calculate Total Speed from active downloads
    useEffect(() => {
        const active = Object.values(downloads).filter(d => d.status === 'downloading');
        const speed = active.reduce((acc, d) => acc + (d.speed_MBs || 0), 0);
        setTotalSpeed(speed);
    }, [downloads]);

    return {
        downloads,
        addDownload,
        openFolder,
        openFile,
        runSpeedTest,
        getLifetimeStats,
        reorderDownload,
        setPriority,
        dailyData,
        analyticsData,
        diskUsage,
        totalSpeed
    };
}
