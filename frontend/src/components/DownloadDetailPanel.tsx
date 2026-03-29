import React from 'react';
import { DownloadItem } from '../types';
import { X, Copy, Check, FolderOpen } from 'lucide-react';
import { cn } from '../utils';
import * as App from '../../wailsjs/go/app/App';

const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const units = ['B', 'kB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1000));
    const val = bytes / Math.pow(1000, i);
    return `${parseFloat(val.toFixed(1))} ${units[i]}`;
};

interface DownloadDetailPanelProps {
    item: DownloadItem;
    onClose: () => void;
}

const formatDateTime = (iso?: string): string => {
    if (!iso) return '-';
    try {
        return new Date(iso).toLocaleString(undefined, {
            year: 'numeric', month: 'short', day: 'numeric',
            hour: '2-digit', minute: '2-digit', second: '2-digit',
        });
    } catch { return iso; }
};

const formatDuration = (seconds?: number): string => {
    if (seconds == null || seconds <= 0) return '-';
    if (seconds < 60) return `${seconds.toFixed(1)}s`;
    const m = Math.floor(seconds / 60);
    const s = Math.round(seconds % 60);
    if (m < 60) return `${m}m ${s}s`;
    const h = Math.floor(m / 60);
    return `${h}h ${m % 60}m ${s}s`;
};

const formatEtaDetail = (item: DownloadItem): string => {
    if (item.status !== 'downloading') return '-';
    if (!item.speed_MBs || item.speed_MBs <= 0) return 'Calculating...';
    const remaining = item.size * (1 - item.progress / 100);
    const seconds = remaining / (item.speed_MBs * 1024 * 1024);
    if (seconds < 60) return `${Math.ceil(seconds)}s`;
    const m = Math.floor(seconds / 60);
    const s = Math.round(seconds % 60);
    if (m < 60) return `${m}m ${s}s`;
    const h = Math.floor(m / 60);
    return `${h}h ${m % 60}m`;
};

const categoryLabels: Record<string, string> = {
    video: 'Video',
    compressed: 'Archive',
    document: 'Document',
    program: 'Program',
    audio: 'Audio',
    image: 'Image',
    other: 'Other',
};

export const DownloadDetailPanel: React.FC<DownloadDetailPanelProps> = ({ item, onClose }) => {
    const [copied, setCopied] = React.useState(false);
    const [copiedPath, setCopiedPath] = React.useState(false);

    const copyUrl = () => {
        navigator.clipboard.writeText(item.url);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
    };

    const copyPath = () => {
        if (!item.path) return;
        navigator.clipboard.writeText(item.path);
        setCopiedPath(true);
        setTimeout(() => setCopiedPath(false), 1500);
    };

    const openInExplorer = () => {
        if (item.path && App && App.OpenFolderByPath) {
            App.OpenFolderByPath(item.path);
        }
    };

    const isActive = item.status === 'downloading' || item.status === 'probing' || item.status === 'merging' || item.status === 'verifying';
    const isPaused = item.status === 'paused';
    const downloaded = item.downloaded || 0;
    const remaining = item.size > 0 ? Math.max(0, item.size - downloaded) : 0;

    const rows: { label: string; value: React.ReactNode }[] = [
        {
            label: 'Source URL',
            value: (
                <div className="flex items-center gap-1.5 min-w-0">
                    <span className="text-[12px] text-th-accent-t truncate font-mono" title={item.url}>{item.url}</span>
                    <button onClick={copyUrl} className="shrink-0 p-0.5 rounded hover:bg-th-raised transition-colors" title="Copy URL">
                        {copied ? <Check size={12} className="text-green-400" /> : <Copy size={12} className="text-th-text-s" />}
                    </button>
                </div>
            ),
        },
        { label: 'File Name', value: <span className="text-[12px] text-th-text truncate">{item.filename}</span> },
        {
            label: 'Size',
            value: (
                <span className="text-[12px] text-th-text font-mono">
                    {item.size ? (
                        (isActive || isPaused) && downloaded > 0
                            ? <>{formatBytes(downloaded)} <span className="text-th-text-s">/</span> {formatBytes(item.size)}</>
                            : formatBytes(item.size)
                    ) : 'Unknown'}
                </span>
            ),
        },
        ...(isActive || isPaused ? [
            { label: 'Remaining', value: <span className="text-[12px] text-th-text font-mono">{item.size > 0 ? formatBytes(remaining) : '-'}</span> },
            { label: 'Speed', value: <span className="text-[12px] text-th-text font-mono">{item.speed_MBs ? `${item.speed_MBs.toFixed(1)} MB/s` : isPaused ? 'Paused' : '-'}</span> },
            { label: 'ETA', value: <span className="text-[12px] text-th-text font-mono">{isPaused ? 'Paused' : formatEtaDetail(item)}</span> },
        ] : []),
        { label: 'Download Type', value: <span className="text-[12px] text-th-text capitalize">{categoryLabels[item.category || ''] || item.category || '-'}</span> },
        { label: 'Resume Support', value: <ResumeTag value={item.accept_ranges} /> },
        { label: 'Started', value: <span className="text-[12px] text-th-text">{formatDateTime(item.started_at || item.created_at)}</span> },
        ...(item.status === 'completed' ? [
            { label: 'Completed', value: <span className="text-[12px] text-th-text">{formatDateTime(item.completed_at)}</span> },
            { label: 'Time Taken', value: <span className="text-[12px] text-th-text font-mono">{formatDuration(item.elapsed)}</span> },
            { label: 'Avg Speed', value: <span className="text-[12px] text-th-text font-mono">{item.avg_speed ? `${formatBytes(item.avg_speed)}/s` : '-'}</span> },
        ] : []),
        {
            label: 'Save Path',
            value: (
                <div className="flex items-center gap-1.5 min-w-0">
                    <span
                        className="text-[12px] text-th-text-s truncate font-mono cursor-pointer hover:text-th-accent-t transition-colors"
                        title={item.path ? `Click to show in explorer: ${item.path}` : undefined}
                        onClick={openInExplorer}
                    >
                        {item.path || '-'}
                    </span>
                    {item.path && (
                        <>
                            <button onClick={copyPath} className="shrink-0 p-0.5 rounded hover:bg-th-raised transition-colors" title="Copy path">
                                {copiedPath ? <Check size={12} className="text-green-400" /> : <Copy size={12} className="text-th-text-s" />}
                            </button>
                            <button onClick={openInExplorer} className="shrink-0 p-0.5 rounded hover:bg-th-raised transition-colors" title="Show in explorer">
                                <FolderOpen size={12} className="text-th-text-s" />
                            </button>
                        </>
                    )}
                </div>
            ),
        },
    ];

    return (
        <div className="shrink-0 border-t border-th-border bg-th-surface/95 backdrop-blur-sm">
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-1.5 border-b border-th-border/50">
                <span className="text-[11px] font-bold text-th-text-m uppercase tracking-wider">Download Details</span>
                <button onClick={onClose} className="p-0.5 rounded hover:bg-th-raised transition-colors text-th-text-s hover:text-th-text">
                    <X size={14} />
                </button>
            </div>
            {/* Content grid */}
            <div className="grid grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-1.5 px-4 py-2.5">
                {rows.map(row => (
                    <div key={row.label} className={cn("min-w-0", row.label === 'Source URL' || row.label === 'Save Path' ? 'col-span-2 lg:col-span-3' : '')}>
                        <div className="text-[10px] text-th-text-s uppercase tracking-wider mb-0.5">{row.label}</div>
                        <div className="truncate">{row.value}</div>
                    </div>
                ))}
            </div>
        </div>
    );
};

const ResumeTag: React.FC<{ value?: boolean }> = ({ value }) => {
    if (value === true) return <span className="text-[11px] font-semibold text-green-400">Yes (Range Supported)</span>;
    if (value === false) return <span className="text-[11px] font-semibold text-yellow-400">No (Single Stream)</span>;
    return <span className="text-[11px] text-th-text-s">Unknown</span>;
};
