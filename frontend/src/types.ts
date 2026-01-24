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
    priority?: number; // 0=Low, 1=Normal, 2=High
    created_at?: string; // Added
};
