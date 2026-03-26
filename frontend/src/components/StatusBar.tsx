import React from 'react';
import { Download, Clock, Database, Activity } from 'lucide-react';
import { cn } from '../utils';

interface StatusBarProps {
    activeDownloads: number;
    pendingDownloads: number;
    dailyData: string;
    globalSpeed: number;
    sidebarCollapsed?: boolean;
}

export const StatusBar: React.FC<StatusBarProps> = ({ activeDownloads, pendingDownloads, dailyData, globalSpeed, sidebarCollapsed = false }) => {
    return (
        <footer className={cn(
            "h-7 fixed bottom-0 right-0 bg-th-surface/90 backdrop-blur-sm border-t border-th-border z-40 flex items-center px-4 gap-6 text-[11px] text-th-text-s transition-all duration-300",
            sidebarCollapsed ? "left-16" : "left-64"
        )}>
            <div className="flex items-center gap-1.5">
                <Download size={11} className={activeDownloads > 0 ? "text-th-accent-t" : "text-th-text-m"} />
                <span>Active: <span className="text-th-text font-medium">{activeDownloads}</span></span>
            </div>
            <div className="flex items-center gap-1.5">
                <Clock size={11} className={pendingDownloads > 0 ? "text-indigo-400" : "text-th-text-m"} />
                <span>Queued: <span className="text-th-text font-medium">{pendingDownloads}</span></span>
            </div>
            <div className="flex items-center gap-1.5">
                <Database size={11} className="text-th-text-m" />
                <span>Today: <span className="text-th-text font-medium">{dailyData}</span></span>
            </div>
            <div className="ml-auto flex items-center gap-1.5">
                <Activity size={11} className={globalSpeed > 0 ? "text-th-accent-t" : "text-th-text-m"} />
                <span className="font-mono"><span className="text-th-text font-medium">{globalSpeed.toFixed(1)}</span> MB/s</span>
            </div>
        </footer>
    );
};
