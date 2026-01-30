export type DownloadItem = {
    id: string;
    url: string; // Added
    filename: string;
    progress: number;
    size: number; // Added
    speed_MBs?: number; // Optional as it's realtime only
    eta?: string; // Optional
    status: "downloading" | "paused" | "completed" | "error" | "pending";
    error?: string;
    path?: string;
    priority?: number; // 0=Low, 1=Normal, 2=High
    queue_order?: number; // Added for sequential ordering
    created_at?: string; // Added
    file_exists?: boolean; // Added
};
