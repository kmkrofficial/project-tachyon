import React, { useMemo } from 'react';
import { Plus, Play, Pause, Clock, Trash2 } from 'lucide-react';
import { cn } from '../utils';
import { useSettingsStore } from '../store';
import { Dropdown } from './common/Dropdown';

const pageTitles: Record<string, string> = {
    all: 'Dashboard',
    analytics: 'Analytics',
    scheduler: 'Scheduler',
    speedtest: 'Speed Test',
};

const hourOptions = Array.from({ length: 24 }, (_, i) => ({
    value: String(i),
    label: String(i).padStart(2, '0'),
}));

const minuteOptions = Array.from({ length: 12 }, (_, i) => ({
    value: String(i * 5),
    label: String(i * 5).padStart(2, '0'),
}));

interface HeaderProps {
    activeTab: string;
    onAddDownload: () => void;
    onPauseAll: () => void;
    onResumeAll: () => void;
    onClear?: () => void;
    sidebarCollapsed?: boolean;
}

export const Header: React.FC<HeaderProps> = ({ activeTab, onAddDownload, onPauseAll, onResumeAll, onClear, sidebarCollapsed = false }) => {
    const isDashboard = activeTab === 'all';
    const isScheduler = activeTab === 'scheduler';
    const schedulerTime = useSettingsStore(s => s.schedulerTime);
    const setSchedulerTime = useSettingsStore(s => s.setSchedulerTime);

    const [hour, minute] = useMemo(() => {
        const parts = (schedulerTime || '02:00').split(':');
        return [parseInt(parts[0]) || 0, parseInt(parts[1]) || 0];
    }, [schedulerTime]);

    const updateTime = (h: number, m: number) => {
        setSchedulerTime(`${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`);
    };

    return (
        <header className={cn(
            "h-16 fixed top-0 right-0 bg-th-surface/80 backdrop-blur-md border-b border-th-border z-40 flex items-center justify-between px-6 dragging-header transition-all duration-300",
            sidebarCollapsed ? "left-16" : "left-64"
        )}>
            {/* Left: Page Title */}
            <h2 className="text-lg font-semibold text-th-text no-drag">{pageTitles[activeTab] ?? activeTab}</h2>

            {/* Spacer for dragging area */}
            <div className="flex-1 dragging-header" />

            {/* Right: Global Actions (Dashboard) */}
            {isDashboard && (
                <div className="flex items-center gap-2 sm:gap-4 no-drag">
                    <button
                        onClick={onResumeAll}
                        className="p-2 text-th-text-s hover:text-th-text hover:bg-th-raised rounded-lg transition-colors"
                        title="Resume All"
                    >
                        <Play size={18} />
                    </button>
                    <button
                        onClick={onPauseAll}
                        className="p-2 text-th-text-s hover:text-th-text hover:bg-th-raised rounded-lg transition-colors"
                        title="Pause All"
                    >
                        <Pause size={18} />
                    </button>
                    <button
                        onClick={onClear}
                        className="p-2 text-th-text-s hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                        title="Clear Downloads"
                    >
                        <Trash2 size={18} />
                    </button>
                    <button
                        onClick={onAddDownload}
                        className="flex items-center gap-2 bg-th-accent hover:bg-th-accent-h text-white px-4 py-2 rounded-lg font-medium shadow-lg shadow-th-accent/20 transition-all active:scale-95"
                    >
                        <Plus size={18} />
                        <span className="text-sm hidden sm:inline">Add Download</span>
                    </button>
                </div>
            )}

            {/* Right: Scheduler Actions */}
            {isScheduler && (
                <div className="flex items-center gap-2 sm:gap-4 no-drag">
                    <div className="flex items-center gap-1.5 bg-th-raised border border-th-border rounded-lg px-2.5 py-1">
                        <Clock size={14} className="text-purple-400 shrink-0" />
                        <span className="text-xs text-th-text-s mr-1">Daily at</span>
                        <Dropdown
                            value={String(hour)}
                            onChange={v => updateTime(parseInt(v), minute)}
                            options={hourOptions}
                            className="w-16"
                        />
                        <span className="text-sm font-mono font-semibold text-th-text-s">:</span>
                        <Dropdown
                            value={String(minute)}
                            onChange={v => updateTime(hour, parseInt(v))}
                            options={minuteOptions}
                            className="w-16"
                        />
                    </div>
                    <button
                        onClick={onClear}
                        className="p-2 text-th-text-s hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                        title="Clear Downloads"
                    >
                        <Trash2 size={18} />
                    </button>
                    <button
                        onClick={onAddDownload}
                        className="flex items-center gap-2 bg-th-accent hover:bg-th-accent-h text-white px-4 py-2 rounded-lg font-medium shadow-lg shadow-th-accent/20 transition-all active:scale-95"
                    >
                        <Plus size={18} />
                        <span className="text-sm hidden sm:inline">Add Download</span>
                    </button>
                </div>
            )}
        </header>
    );
};
