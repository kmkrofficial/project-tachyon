import React from 'react';
import { Plus, Play, Pause, Activity } from 'lucide-react';
import { cn } from '../utils';

interface HeaderProps {
    onAddDownload: () => void;
    onPauseAll: () => void;
    onResumeAll: () => void;
    globalSpeed?: number; // MB/s
}

export const Header: React.FC<HeaderProps> = ({ onAddDownload, onPauseAll, onResumeAll, globalSpeed = 0 }) => {
    return (
        <header className="h-16 fixed top-0 right-0 left-64 bg-slate-900/80 backdrop-blur-md border-b border-slate-800 z-40 flex items-center justify-between px-6 dragging-header">
            {/* Left: Breadcrumbs / Title */}
            <div className="flex items-center gap-2">
                <h1 className="text-lg font-semibold text-slate-100">Dashboard</h1>
                <span className="text-slate-600">/</span>
                <span className="text-sm text-slate-400">Overview</span>
            </div>

            {/* Right: Global Actions */}
            <div className="flex items-center gap-4 no-drag">

                {/* Global Speed Indicator */}
                <div className="flex items-center gap-2 px-3 py-1.5 bg-slate-800/50 rounded-full border border-slate-700/50" title="Total Real-time Bandwidth">
                    <Activity size={14} className="text-cyan-400 animate-pulse" />
                    <span className="text-sm font-mono text-cyan-400">{globalSpeed.toFixed(1)} MB/s</span>
                </div>

                <div className="h-6 w-px bg-slate-800"></div>

                {/* Pause/Resume All */}
                <button
                    onClick={onResumeAll}
                    className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
                    title="Resume All"
                >
                    <Play size={18} />
                </button>
                <button
                    onClick={onPauseAll}
                    className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
                    title="Pause All"
                >
                    <Pause size={18} />
                </button>

                {/* Add Download Button */}
                <button
                    onClick={onAddDownload}
                    className="flex items-center gap-2 bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white px-4 py-2 rounded-lg font-medium shadow-lg shadow-cyan-900/20 transition-all active:scale-95"
                >
                    <Plus size={18} />
                    <span className="text-sm">Add Download</span>
                </button>
            </div>
        </header>
    );
};
