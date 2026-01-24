import { File, FileVideo, FileArchive, FileText, Pause, Play, X, FolderOpen } from "lucide-react";
import { ProgressBar } from "./ProgressBar";
import { DownloadItem as DownloadItemType } from "../types";

type DownloadItemProps = {
    item: DownloadItemType;
    onOpenFolder?: (path: string) => void;
};

export function DownloadItem({ item, onOpenFolder }: DownloadItemProps) {
    const getIcon = (filename: string) => {
        if (filename.endsWith(".mp4") || filename.endsWith(".mkv")) return <FileVideo className="text-purple-400" size={32} />;
        if (filename.endsWith(".zip") || filename.endsWith(".rar")) return <FileArchive className="text-yellow-400" size={32} />;
        if (filename.endsWith(".txt") || filename.endsWith(".md")) return <FileText className="text-gray-400" size={32} />;
        return <File className="text-blue-400" size={32} />;
    };

    return (
        <div className="bg-gray-800 rounded-xl p-4 flex items-center gap-4 shadow-sm border border-gray-700 hover:border-gray-600 transition-colors">
            <div className="bg-gray-900 p-3 rounded-lg">
                {getIcon(item.filename)}
            </div>

            <div className="flex-1 min-w-0">
                <div className="flex justify-between items-center mb-1">
                    <h3 className="text-white font-medium truncate" title={item.filename}>
                        {item.filename}
                    </h3>
                    <span className="text-xs text-gray-400 font-mono">
                        {item.status === "downloading" && `${item.speed_MBs ? item.speed_MBs.toFixed(1) : "0.0"} MB/s • ETA: ${item.eta || "--"}`}
                        {item.status === "completed" && "Completed"}
                        {item.status === "error" && "Error"}
                    </span>
                </div>

                <ProgressBar progress={item.progress} status={item.status} />

                <div className="mt-1 flex justify-between text-xs text-gray-500">
                    <span>{item.progress.toFixed(1)}%</span>
                    {item.error && <span className="text-red-400">{item.error}</span>}
                </div>
            </div>

            <div className="flex gap-2">
                <button className="p-2 hover:bg-gray-700 rounded-full text-gray-400 hover:text-white transition-colors" title="Pause/Resume">
                    {item.status === "paused" ? <Play size={18} /> : <Pause size={18} />}
                </button>
                <button className="p-2 hover:bg-gray-700 rounded-full text-gray-400 hover:text-white transition-colors" title="Cancel">
                    <X size={18} />
                </button>
                <button
                    className="p-2 hover:bg-gray-700 rounded-full text-gray-400 hover:text-white transition-colors"
                    title="Open Folder"
                    onClick={() => onOpenFolder && item.path && onOpenFolder(item.path)}
                >
                    <FolderOpen size={18} />
                </button>
            </div>
        </div>
    );
}
