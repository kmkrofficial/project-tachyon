import React, { useState, useEffect } from 'react';
import { Plus, Play, Pause, Activity, Wifi, Sun, Moon, Monitor } from 'lucide-react';
import { cn } from '../utils';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';
import { useSettingsStore } from '../store';

interface HeaderProps {
    onAddDownload: () => void;
    onPauseAll: () => void;
    onResumeAll: () => void;
    globalSpeed?: number; // MB/s
    sidebarCollapsed?: boolean;
}

interface NetworkHealthEvent {
    level: string; // "normal", "stressed", "critical"
    details?: string;
}

const themeOptions = [
    { value: 'light' as const, icon: Sun, label: 'Light' },
    { value: 'dark' as const, icon: Moon, label: 'Dark' },
    { value: 'system' as const, icon: Monitor, label: 'System' },
];

export const Header: React.FC<HeaderProps> = ({ onAddDownload, onPauseAll, onResumeAll, globalSpeed = 0, sidebarCollapsed = false }) => {
    const [networkHealth, setNetworkHealth] = useState<NetworkHealthEvent>({ level: 'normal' });
    const theme = useSettingsStore(s => s.theme);
    const setTheme = useSettingsStore(s => s.setTheme);

    // Listen for network health updates
    useEffect(() => {
        const handleNetworkHealth = (event: NetworkHealthEvent) => {
            setNetworkHealth(event);
        };

        EventsOn("network:congestion_level", handleNetworkHealth);

        // Poll initial network health
        if (window.go?.app?.App?.GetNetworkHealth) {
            window.go.app.App.GetNetworkHealth().then(setNetworkHealth).catch(console.error);
        }

        return () => {
            EventsOff("network:congestion_level");
        };
    }, []);

    const getHealthColor = () => {
        switch (networkHealth.level) {
            case 'critical': return 'bg-red-500';
            case 'stressed': return 'bg-yellow-500';
            default: return 'bg-green-500';
        }
    };

    const getHealthLabel = () => {
        switch (networkHealth.level) {
            case 'critical': return 'Critical';
            case 'stressed': return 'Stressed';
            default: return 'Healthy';
        }
    };

    const cycleTheme = () => {
        const order: Array<'light' | 'dark' | 'system'> = ['light', 'dark', 'system'];
        const idx = order.indexOf(theme);
        setTheme(order[(idx + 1) % order.length]);
    };

    const currentThemeIcon = themeOptions.find(t => t.value === theme)!;
    const ThemeIcon = currentThemeIcon.icon;

    return (
        <header className={cn(
            "h-16 fixed top-0 right-0 bg-th-surface/80 backdrop-blur-md border-b border-th-border z-40 flex items-center justify-between px-6 dragging-header transition-all duration-300",
            sidebarCollapsed ? "left-16" : "left-64"
        )}>
            {/* Left: Breadcrumbs / Title */}
            <div className="flex items-center gap-2">
                <h1 className="text-lg font-semibold text-th-text">Dashboard</h1>
                <span className="text-th-text-m">/</span>
                <span className="text-sm text-th-text-s">Overview</span>
            </div>

            {/* Right: Global Actions */}
            <div className="flex items-center gap-2 sm:gap-4 no-drag">

                {/* Network Health Indicator */}
                <div
                    className="hidden md:flex items-center gap-2 px-3 py-1.5 bg-th-raised/50 rounded-full border border-th-border-s/50 cursor-help"
                    title={`Network: ${getHealthLabel()}${networkHealth.details ? ` - ${networkHealth.details}` : ''}`}
                >
                    <Wifi size={14} className="text-th-text-s" />
                    <div className={cn("w-2 h-2 rounded-full animate-pulse", getHealthColor())} />
                    <span className="text-xs text-th-text-s">{getHealthLabel()}</span>
                </div>

                {/* Global Speed Indicator */}
                <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 bg-th-raised/50 rounded-full border border-th-border-s/50" title="Total Real-time Bandwidth">
                    <Activity size={14} className="text-cyan-500 animate-pulse" />
                    <span className="text-sm font-mono text-cyan-500">{globalSpeed.toFixed(1)} MB/s</span>
                </div>

                <div className="h-6 w-px bg-th-border hidden sm:block"></div>

                {/* Theme Toggle */}
                <button
                    onClick={cycleTheme}
                    className="p-2 text-th-text-s hover:text-th-text hover:bg-th-raised rounded-lg transition-colors"
                    title={`Theme: ${currentThemeIcon.label}`}
                >
                    <ThemeIcon size={18} />
                </button>

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
                    className="flex items-center gap-2 bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white px-4 py-2 rounded-lg font-medium shadow-lg shadow-cyan-900/20 transition-all active:scale-95"
                >
                    <Plus size={18} />
                    <span className="text-sm hidden sm:inline">Add Download</span>
                </button>
            </div>
        </header>
    );
};
