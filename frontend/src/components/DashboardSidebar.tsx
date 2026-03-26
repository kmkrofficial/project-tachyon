import React from 'react';
import { Download, CheckCircle, Pause, AlertCircle, Clock, FileVideo, FileArchive, File, FileText, Cpu } from 'lucide-react';
import { cn } from '../utils';

export type StatusFilter = 'all' | 'downloading' | 'completed' | 'paused' | 'pending' | 'error';
export type CategoryFilter = 'all' | 'video' | 'compressed' | 'document' | 'program' | 'other';

interface DashboardSidebarProps {
    statusFilter: StatusFilter;
    categoryFilter: CategoryFilter;
    onStatusChange: (s: StatusFilter) => void;
    onCategoryChange: (c: CategoryFilter) => void;
    counts: {
        all: number;
        downloading: number;
        completed: number;
        paused: number;
        pending: number;
        error: number;
    };
}

const statusItems: { value: StatusFilter; label: string; Icon: React.FC<any>; color: string }[] = [
    { value: 'all', label: 'All Downloads', Icon: Download, color: 'text-th-text-s' },
    { value: 'downloading', label: 'Active', Icon: Download, color: 'text-th-accent-t' },
    { value: 'completed', label: 'Completed', Icon: CheckCircle, color: 'text-green-400' },
    { value: 'paused', label: 'Paused', Icon: Pause, color: 'text-yellow-400' },
    { value: 'pending', label: 'Queued', Icon: Clock, color: 'text-th-text-s' },
    { value: 'error', label: 'Failed', Icon: AlertCircle, color: 'text-red-400' },
];

const categoryItems: { value: CategoryFilter; label: string; Icon: React.FC<any>; exts: string[] }[] = [
    { value: 'all', label: 'All Types', Icon: File, exts: [] },
    { value: 'video', label: 'Video', Icon: FileVideo, exts: ['mp4', 'mkv', 'webm', 'avi', 'mov', 'flv', 'wmv'] },
    { value: 'compressed', label: 'Archives', Icon: FileArchive, exts: ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz'] },
    { value: 'document', label: 'Documents', Icon: FileText, exts: ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'csv'] },
    { value: 'program', label: 'Programs', Icon: Cpu, exts: ['exe', 'msi', 'dmg', 'deb', 'rpm', 'appimage', 'apk'] },
    { value: 'other', label: 'Other', Icon: File, exts: [] },
];

export const getCategoryExts = (cat: CategoryFilter): string[] => {
    return categoryItems.find(c => c.value === cat)?.exts || [];
};

export const allKnownExts = categoryItems.flatMap(c => c.exts);

export const DashboardSidebar: React.FC<DashboardSidebarProps> = ({
    statusFilter, categoryFilter, onStatusChange, onCategoryChange, counts
}) => {
    return (
        <div className="w-44 shrink-0 overflow-y-auto space-y-5 pr-4 border-r border-th-border scrollbar-thin scrollbar-thumb-th-raised scrollbar-track-transparent pt-0.5">
            {/* Status */}
            <div>
                <h4 className="text-[10px] font-bold text-th-text-m uppercase tracking-widest mb-2 px-2">Status</h4>
                <div className="space-y-0.5">
                    {statusItems.map(({ value, label, Icon, color }) => (
                        <button
                            key={value}
                            onClick={() => onStatusChange(value)}
                            className={cn(
                                "w-full flex items-center gap-2 px-2 py-1.5 rounded-lg text-sm transition-colors",
                                statusFilter === value
                                    ? "bg-th-raised text-th-text font-medium"
                                    : "text-th-text-s hover:text-th-text hover:bg-th-raised/50"
                            )}
                        >
                            <Icon size={14} className={statusFilter === value ? color : ''} />
                            <span className="flex-1 text-left">{label}</span>
                            <span className="text-xs text-th-text-m tabular-nums">
                                {counts[value]}
                            </span>
                        </button>
                    ))}
                </div>
            </div>

            {/* Category */}
            <div>
                <h4 className="text-[10px] font-bold text-th-text-m uppercase tracking-widest mb-2 px-2">File Type</h4>
                <div className="space-y-0.5">
                    {categoryItems.map(({ value, label, Icon }) => (
                        <button
                            key={value}
                            onClick={() => onCategoryChange(value)}
                            className={cn(
                                "w-full flex items-center gap-2 px-2 py-1.5 rounded-lg text-sm transition-colors",
                                categoryFilter === value
                                    ? "bg-th-raised text-th-text font-medium"
                                    : "text-th-text-s hover:text-th-text hover:bg-th-raised/50"
                            )}
                        >
                            <Icon size={14} />
                            <span className="flex-1 text-left">{label}</span>
                        </button>
                    ))}
                </div>
            </div>
        </div>
    );
};
