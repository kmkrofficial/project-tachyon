import React from 'react';
import { Plus, Play, Pause } from 'lucide-react';
import { cn } from '../utils';

const pageTitles: Record<string, string> = {
    all: 'Dashboard',
    analytics: 'Analytics',
    scheduler: 'Scheduler',
    speedtest: 'Speed Test',
};

interface HeaderProps {
    activeTab: string;
    onAddDownload: () => void;
    onPauseAll: () => void;
    onResumeAll: () => void;
    sidebarCollapsed?: boolean;
}

export const Header: React.FC<HeaderProps> = ({ activeTab, onAddDownload, onPauseAll, onResumeAll, sidebarCollapsed = false }) => {
    const isDashboard = activeTab === 'all';

    return (
        <header className={cn(
            "h-16 fixed top-0 right-0 bg-th-surface/80 backdrop-blur-md border-b border-th-border z-40 flex items-center justify-between px-6 dragging-header transition-all duration-300",
            sidebarCollapsed ? "left-16" : "left-64"
        )}>
            {/* Left: Page Title */}
            <h2 className="text-lg font-semibold text-th-text no-drag">{pageTitles[activeTab] ?? activeTab}</h2>

            {/* Spacer for dragging area */}
            <div className="flex-1 dragging-header" />

            {/* Right: Global Actions (Dashboard only) */}
            {isDashboard && (
                <div className="flex items-center gap-2 sm:gap-4 no-drag">

                    {/* Pause/Resume All */}
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

                    {/* Add Download Button */}
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
