import { useState, useCallback, useEffect } from 'react';
import { Search } from 'lucide-react';
import { Sidebar } from './components/Sidebar';
import { Header } from './components/Header';
import { DownloadsTable } from './components/DownloadsTable';
import { AddURLModal } from './components/AddURLModal';
import { SettingsModal } from './components/SettingsModal';
import { AnalyticsTab } from './components/AnalyticsTab';
import { SpeedTestTab } from './components/SpeedTestTab';
import { useTachyon } from './hooks/useTachyon';
import { useTheme } from './hooks/useTheme';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { ToastContainer, ToastMessage } from './components/Toast';
import { EventsOn } from '../wailsjs/runtime/runtime';

function App() {
    const [activeTab, setActiveTab] = useState("all");
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);
    const [toasts, setToasts] = useState<ToastMessage[]>([]);
    const [searchQuery, setSearchQuery] = useState('');
    const [statusFilter, setStatusFilter] = useState<string>('all');

    // Activate theme system
    useTheme();

    const addToast = useCallback((type: 'success' | 'error' | 'warning' | 'info', title: string, message: string) => {
        const id = Math.random().toString(36).substr(2, 9);
        setToasts(prev => [...prev, { id, type, title, message }]);
    }, []);

    const removeToast = useCallback((id: string) => {
        setToasts(prev => prev.filter(t => t.id !== id));
    }, []);

    // Listen to Backend Logs
    useEffect(() => {
        const cleanup = EventsOn("log:entry", (entry: any) => {
            const { level, message, time, data } = entry;
            const style = level === 'ERROR' ? 'color: #ef4444; font-weight: bold;'
                : level === 'WARN' ? 'color: #f59e0b; font-weight: bold;'
                    : 'color: #06b6d4;';
            console.log(`%c[BE] [${level}] ${message}`, style, Object.keys(data).length > 0 ? data : '');
        });
        return cleanup;
    }, []);

    // openFolder and openFile trigger backend ops by ID
    // We pass addToast to useTachyon if we want it to manage some errors, or just pass it down to components
    const { downloads, addDownload, openFolder, openFile, dailyData, diskUsage, totalSpeed, reorderDownload, setPriority } = useTachyon();

    // Keyboard shortcuts
    useKeyboardShortcuts({
        onNewDownload: () => setIsModalOpen(true),
        onPauseResume: () => {
            const active = Object.values(downloads).find(d => d.status === 'downloading');
            if (active) { window.go?.app?.App?.PauseDownload?.(active.id); return; }
            const paused = Object.values(downloads).find(d => d.status === 'paused');
            if (paused) { window.go?.app?.App?.ResumeDownload?.(paused.id); }
        },
        onDelete: () => {
            const first = Object.values(downloads).find(d => ['downloading', 'paused', 'pending', 'error'].includes(d.status));
            if (first) window.go?.app?.App?.DeleteDownload?.(first.id, false);
        },
    });

    // Filter downloads based on active tab, search, and status filter
    const filteredDownloads = Object.values(downloads)
        .filter(item => {
            // Tab filter
            if (activeTab !== "all" && activeTab !== "settings" && activeTab !== "analytics") {
                if (item.status !== activeTab) return false;
            }
            // Status dropdown filter
            if (statusFilter !== 'all' && item.status !== statusFilter) return false;
            // Text search
            if (searchQuery) {
                const q = searchQuery.toLowerCase();
                const matchFile = item.filename?.toLowerCase().includes(q);
                const matchUrl = item.url?.toLowerCase().includes(q);
                if (!matchFile && !matchUrl) return false;
            }
            return true;
        })
        .sort((a, b) => {
            // Sort active/pending items by Queue Order (Ascending)
            const activeStates = ["downloading", "pending", "paused", "stopped"];
            const isActiveA = activeStates.includes(a.status);
            const isActiveB = activeStates.includes(b.status);

            if (isActiveA && isActiveB) {
                return (a.queue_order || 0) - (b.queue_order || 0);
            }

            // Put active items before completed/error items
            if (isActiveA && !isActiveB) return -1;
            if (!isActiveA && isActiveB) return 1;

            // Sort completed/others by Created At Descending
            if (a.created_at && b.created_at) return b.created_at.localeCompare(a.created_at);
            return 0;
        });

    return (
        <div className="flex h-screen bg-slate-950 text-slate-200 font-sans overflow-hidden select-none">

            <ToastContainer toasts={toasts} removeToast={removeToast} />

            {/* Sidebar (Fixed Width) */}
            <Sidebar activeTab={activeTab} setActiveTab={(tab) => {
                if (tab === 'settings') setIsSettingsOpen(true);
                else setActiveTab(tab);
            }} diskUsage={diskUsage} />

            {/* Main Layout (Left Margin for Sidebar) */}
            <div className="flex-1 flex flex-col ml-64 min-w-0 h-full">

                {/* Fixed Header */}
                <Header
                    onAddDownload={() => setIsModalOpen(true)}
                    // @ts-ignore
                    onPauseAll={() => window.go.app.App.PauseAllDownloads()}
                    // @ts-ignore
                    onResumeAll={() => window.go.app.App.ResumeAllDownloads()}
                    globalSpeed={totalSpeed}
                />

                {/* Scrollable Content Area */}
                <main className="flex-1 overflow-y-auto pt-16 bg-slate-950 scrollbar-thin scrollbar-thumb-slate-800 scrollbar-track-transparent">
                    <div className="max-w-[1600px] mx-auto p-6 md:p-8">

                        {/* Dynamic Content */}
                        {activeTab === 'analytics' ? (
                            <AnalyticsTab />
                        ) : activeTab === 'speedtest' ? (
                            <SpeedTestTab />
                        ) : (
                            <div className="space-y-6">
                                {/* Dashboard Widgets (Only on Dashboard) */}
                                {activeTab === 'all' && <DashboardWidgets downloads={Object.values(downloads)} dailyData={dailyData} totalSpeed={totalSpeed} />}

                                {/* Search & Filter Bar */}
                                <div className="flex items-center gap-3">
                                    <div className="flex-1 relative">
                                        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                                        <input
                                            type="text"
                                            placeholder="Search by filename or URL..."
                                            value={searchQuery}
                                            onChange={e => setSearchQuery(e.target.value)}
                                            className="w-full bg-slate-900 border border-slate-800 rounded-lg pl-10 pr-4 py-2 text-sm text-slate-200 placeholder-slate-600 focus:border-cyan-500 focus:outline-none"
                                        />
                                    </div>
                                    <select
                                        value={statusFilter}
                                        onChange={e => setStatusFilter(e.target.value)}
                                        className="bg-slate-900 border border-slate-800 rounded-lg px-3 py-2 text-sm text-slate-200 focus:border-cyan-500 focus:outline-none"
                                    >
                                        <option value="all">All Status</option>
                                        <option value="downloading">Downloading</option>
                                        <option value="completed">Completed</option>
                                        <option value="paused">Paused</option>
                                        <option value="pending">Pending</option>
                                        <option value="error">Error</option>
                                    </select>
                                </div>

                                {/* Data Grid */}
                                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-2xl shadow-black/20">
                                    <DownloadsTable
                                        data={filteredDownloads}
                                        onOpenFile={openFile}
                                        onOpenFolder={openFolder}
                                        onReorder={reorderDownload}
                                        onSetPriority={setPriority}
                                        addToast={addToast}
                                    />
                                </div>
                            </div>
                        )}
                    </div>
                </main>
            </div>

            <AddURLModal
                isOpen={isModalOpen}
                onClose={() => setIsModalOpen(false)}
                onAdd={addDownload}
            />

            <SettingsModal
                isOpen={isSettingsOpen}
                onClose={() => setIsSettingsOpen(false)}
            />
        </div>
    );
}

// Temporary Widget Component
const DashboardWidgets = ({ downloads, dailyData, totalSpeed }: { downloads: any[], dailyData: string, totalSpeed: number }) => {
    const active = downloads.filter(d => d.status === 'downloading').length;
    const pending = downloads.filter(d => d.status === 'pending').length;

    return (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <Widget title="Active Downloads" value={active.toString()} subtitle={`${totalSpeed.toFixed(1)} MB/s Total`} color="cyan" />
            <Widget title="Queue Pending" value={pending.toString()} subtitle="Next: --" color="indigo" />
            <Widget title="Data Today" value={dailyData} subtitle="Daily Usage" color="purple" />
        </div>
    );
}

const Widget = ({ title, value, subtitle, color }: any) => {
    const colors: any = {
        cyan: "border-l-cyan-500 from-cyan-500/10",
        indigo: "border-l-indigo-500 from-indigo-500/10",
        purple: "border-l-purple-500 from-purple-500/10",
    }
    return (
        <div className={`bg-gradient-to-r ${colors[color]} to-transparent bg-slate-900 border-l-4 border-y border-r border-slate-800 p-6 rounded-lg shadow-lg`}>
            <h3 className="text-slate-400 text-sm font-medium uppercase tracking-wider mb-1">{title}</h3>
            <div className="flex items-baseline gap-2">
                <span className="text-3xl font-bold text-white">{value}</span>
                <span className={`text-xs font-mono px-2 py-0.5 rounded-full bg-slate-800 text-slate-300`}>{subtitle}</span>
            </div>
        </div>
    )
}

export default App;
