import { useEffect, useState } from "react";

// Define the shape of a Download Item locally to match Backend events
export type DownloadItem = {
    id: string;
    filename: string;
    progress: number;
    speed_MBs: number;
    eta: string;
    status: "downloading" | "paused" | "completed" | "error";
    error?: string;
    path?: string;
};

// Declare the Wails runtime/window object types if not already present
declare global {
    interface Window {
        runtime: {
            EventsOn: (event: string, callback: (data: any) => void) => void;
            EventsOff: (event: string) => void;
        };
        go: {
            main: {
                App: {
                    AddDownload: (url: string) => Promise<string>;
                };
            };
        };
    }
}

export function useTachyon() {
    const [downloads, setDownloads] = useState<Record<string, DownloadItem>>({});

    useEffect(() => {
        // Check if Wails runtime is available
        if (!window.runtime) {
            console.warn("Wails runtime not found. Are you running in a browser?");
            return;
        }

        // Listen for Progress
        window.runtime.EventsOn("download:progress", (data: any) => {
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
        window.runtime.EventsOn("download:completed", (data: any) => {
            setDownloads((prev) => ({
                ...prev,
                [data.id]: {
                    ...prev[data.id],
                    status: "completed",
                    progress: 100,
                    path: data.path
                }
            }))
        })

        // Listen for Errors
        window.runtime.EventsOn("download:failed", (data: any) => {
            setDownloads((prev) => ({
                ...prev,
                [data.id]: {
                    ...prev[data.id],
                    status: "error",
                    error: data.error
                }
            }))
        })

        return () => {
            // Cleanup listeners if necessary (Wails handles usually persist, but good practice)
            // window.runtime.EventsOff("download:progress");
        };
    }, []);

    const addDownload = async (url: string) => {
        if (window.go?.main?.App?.AddDownload) {
            try {
                const id = await window.go.main.App.AddDownload(url);
                // Initialize the item immediately in the UI
                setDownloads(prev => ({
                    ...prev,
                    [id]: {
                        id,
                        filename: "Initializing...",
                        progress: 0,
                        speed_MBs: 0,
                        eta: "--",
                        status: "downloading"
                    }
                }))
                return id;
            } catch (e) {
                console.error("Failed to add download", e);
                throw e;
            }
        } else {
            console.warn("Backend not connected");
        }
    };

    return {
        downloads,
        addDownload,
    };
}
