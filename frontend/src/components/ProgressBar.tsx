import clsx from "clsx";

type ProgressBarProps = {
    progress: number; // 0 to 100
    status?: "downloading" | "completed" | "error" | "paused";
};

export function ProgressBar({ progress, status = "downloading" }: ProgressBarProps) {
    const getColor = () => {
        switch (status) {
            case "completed": return "bg-green-500";
            case "error": return "bg-red-500";
            case "paused": return "bg-yellow-500";
            default: return "bg-blue-500";
        }
    };

    return (
        <div className="h-2 w-full bg-gray-700 rounded-full overflow-hidden">
            <div
                className={clsx("h-full transition-all duration-300 ease-out", getColor())}
                style={{ width: `${Math.max(0, Math.min(100, progress))}%` }}
            />
        </div>
    );
}
