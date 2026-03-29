import React from 'react';
import { DownloadItem } from '../types';
import { X, Copy, Check } from 'lucide-react';
import { cn } from '../utils';

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

    const copyUrl = () => {
        navigator.clipboard.writeText(item.url);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
    };

    const isActive = item.status === 'downloading' || item.status === 'probing';
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
            { label: 'ETA', value: <span className="text-[12px] text-th-text font-mono">{item.eta && item.eta !== '--' ? item.eta : isPaused ? 'Paused' : '-'}</span> },
        ] : []),
        { label: 'Download Type', value: <span className="text-[12px] text-th-text capitalize">{categoryLabels[item.category || ''] || item.category || '-'}</span> },
        { label: 'Priority', value: <PriorityTag value={item.priority} /> },
        { label: 'Resume Support', value: <ResumeTag value={item.accept_ranges} /> },
        { label: 'Started', value: <span className="text-[12px] text-th-text">{formatDateTime(item.started_at || item.created_at)}</span> },
        ...(item.status === 'completed' ? [
            { label: 'Completed', value: <span className="text-[12px] text-th-text">{formatDateTime(item.completed_at)}</span> },
            { label: 'Time Taken', value: <span className="text-[12px] text-th-text font-mono">{formatDuration(item.elapsed)}</span> },
            { label: 'Avg Speed', value: <span className="text-[12px] text-th-text font-mono">{item.avg_speed ? `${formatBytes(item.avg_speed)}/s` : '-'}</span> },
        ] : []),
        {
            label: 'Save Path',
            value: <span className="text-[12px] text-th-text-s truncate font-mono" title={item.path}>{item.path || '-'}</span>,
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

const PriorityTag: React.FC<{ value?: number }> = ({ value }) => {
    if (value === 3) return <span className="text-[11px] font-semibold text-red-400">High</span>;
    if (value === 1) return <span className="text-[11px] font-semibold text-blue-400">Low</span>;
    return <span className="text-[11px] font-semibold text-th-text">Normal</span>;
};

const ResumeTag: React.FC<{ value?: boolean }> = ({ value }) => {
    if (value === true) return <span className="text-[11px] font-semibold text-green-400">Yes (Range Supported)</span>;
    if (value === false) return <span className="text-[11px] font-semibold text-yellow-400">No (Single Stream)</span>;
    return <span className="text-[11px] text-th-text-s">Unknown</span>;
};
