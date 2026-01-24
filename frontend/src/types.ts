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
