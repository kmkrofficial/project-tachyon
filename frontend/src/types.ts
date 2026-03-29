export type DownloadItem = {
    id: string;
    url: string;
    filename: string;
    progress: number;
    size: number;
    downloaded?: number;
    speed_MBs?: number;
    eta?: string;
    status: "downloading" | "paused" | "completed" | "error" | "pending" | "probing" | "merging" | "verifying";
    error?: string;
    path?: string;
    queue_order?: number;
    created_at?: string;
    file_exists?: boolean;
    // Detail panel fields
    category?: string;
    accept_ranges?: boolean;
    started_at?: string;
    completed_at?: string;
    elapsed?: number;        // seconds
    avg_speed?: number;      // bytes/sec
};
