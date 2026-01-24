import { useEffect, useState } from "react";
// @ts-ignore
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
// @ts-ignore
import * as App from "../../wailsjs/go/main/App";

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

export function useTachyon() {
    const [downloads, setDownloads] = useState<Record<string, DownloadItem>>({});

    useEffect(() => {
        // Listen for Progress
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
            // EventsOff("download:progress"); 
            // Wails events usually stick around, but if we wanted to unregister:
            // EventsOff("download:progress");
        };
    }, []);

    const addDownload = async (url: string) => {
        if (App && App.AddDownload) {
            try {
                const id = await App.AddDownload(url);
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
            } catch (e: any) {
                console.error("Failed to add download", e);
                throw e;
            }
        } else {
            console.warn("Backend not connected");
            throw new Error("Backend not connected");
        }
    };

    return {
        downloads,
        addDownload,
    };
}
