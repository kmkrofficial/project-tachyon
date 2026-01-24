export type DownloadItem = {
    id: string;
    url: string; // Added
    filename: string;
    progress: number;
    size: number; // Added
    speed_MBs?: number; // Optional as it's realtime only
    eta?: string; // Optional
    status: "downloading" | "paused" | "completed" | "error";
    error?: string;
    path?: string;
    created_at: string; // Added
};
