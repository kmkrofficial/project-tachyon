import { useState, useCallback, useEffect } from 'react';
import { Search } from 'lucide-react';
import { Sidebar } from './components/Sidebar';
import { Header } from './components/Header';
import { DownloadsTable } from './components/DownloadsTable';
import { AddURLModal } from './components/AddURLModal';
import { SettingsModal } from './components/SettingsModal';
import { AnalyticsTab } from './components/AnalyticsTab';
import { SpeedTestTab } from './components/SpeedTestTab';
import { StatusBar } from './components/StatusBar';
import { useTachyon } from './hooks/useTachyon';
import { useTheme } from './hooks/useTheme';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';
import { ToastContainer, ToastMessage } from './components/Toast';
import { EventsOn } from '../wailsjs/runtime/runtime';
import * as AppBinding from '../wailsjs/go/app/App';
import { useSettingsStore } from './store';
import { cn } from './utils';
import { StatusFilter, CategoryFilter, getCategoryExts, allKnownExts } from './components/DashboardSidebar';
import { Dropdown } from './components/common/Dropdown';

function App() {
    const [activeTab, setActiveTab] = useState("all");
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);
    const [pasteUrl, setPasteUrl] = useState<string | undefined>(undefined);
    const [toasts, setToasts] = useState<ToastMessage[]>([]);
    const [searchQuery, setSearchQuery] = useState('');
    const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
    const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all');
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

    // Activate theme system
    useTheme();

    const sidebarCollapsed = useSettingsStore(s => s.sidebarCollapsed);
    const setSidebarCollapsed = useSettingsStore(s => s.setSidebarCollapsed);

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
    const { downloads, addDownload, openFolder, openFile, dailyData, totalSpeed, reorderDownload, setPriority } = useTachyon();

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

    const isValidUrl = useCallback((text: string) => /^https?:\/\/.+/i.test(text.trim()), []);

    const quickDownload = useSettingsStore(s => s.quickDownload);
    const downloadPath = useSettingsStore(s => s.downloadPath);

    // Global Ctrl+V paste handler
    useEffect(() => {
        const handlePaste = (e: ClipboardEvent) => {
            // Don't intercept if user is typing in an input/textarea
            const tag = (e.target as HTMLElement)?.tagName;
            if (tag === 'INPUT' || tag === 'TEXTAREA') return;
            const text = e.clipboardData?.getData('text')?.trim();
            if (text && isValidUrl(text) && !isModalOpen) {
                if (quickDownload) {
                    addDownload(text, undefined, undefined, downloadPath || undefined);
                } else {
                    setPasteUrl(text);
                    setIsModalOpen(true);
                }
            }
        };
        window.addEventListener('paste', handlePaste);
        return () => window.removeEventListener('paste', handlePaste);
    }, [isModalOpen, isValidUrl, quickDownload, downloadPath, addDownload]);

    // Global drag-and-drop handler
    useEffect(() => {
        const handleDragOver = (e: DragEvent) => {
            e.preventDefault();
            if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy';
        };
        const handleDrop = (e: DragEvent) => {
            e.preventDefault();
            const text = e.dataTransfer?.getData('text/uri-list') || e.dataTransfer?.getData('text/plain') || '';
            const trimmed = text.trim();
            if (trimmed && isValidUrl(trimmed) && !isModalOpen) {
                if (quickDownload) {
                    addDownload(trimmed, undefined, undefined, downloadPath || undefined);
                } else {
                    setPasteUrl(trimmed);
                    setIsModalOpen(true);
                }
            }
        };
        window.addEventListener('dragover', handleDragOver);
        window.addEventListener('drop', handleDrop);
        return () => {
            window.removeEventListener('dragover', handleDragOver);
            window.removeEventListener('drop', handleDrop);
        };
    }, [isModalOpen, isValidUrl, quickDownload, downloadPath, addDownload]);

    // Filter downloads based on active tab, search, status and category
    const allDownloads = Object.values(downloads);
    const filteredDownloads = allDownloads
        .filter(item => {
            // Tab filter
            if (activeTab !== "all" && activeTab !== "settings" && activeTab !== "analytics") {
                if (item.status !== activeTab) return false;
            }
            // Status sidebar filter
            if (statusFilter !== 'all') {
                // Treat "probing" as "downloading" for filter purposes
                const effectiveStatus = item.status === 'probing' ? 'downloading' : item.status;
                if (effectiveStatus !== statusFilter) return false;
            }
            // Category filter
            if (categoryFilter !== 'all') {
                const ext = item.filename?.split('.').pop()?.toLowerCase() || '';
                if (categoryFilter === 'other') {
                    if (allKnownExts.includes(ext)) return false;
                } else {
                    const exts = getCategoryExts(categoryFilter);
                    if (!exts.includes(ext)) return false;
                }
            }
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
            const activeStates = ["downloading", "probing", "pending", "paused", "stopped"];
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
        <div className="flex h-screen bg-th-base text-th-text font-sans overflow-hidden select-none">

            <ToastContainer toasts={toasts} removeToast={removeToast} />

            {/* Sidebar (Fixed Width) */}
            <Sidebar activeTab={activeTab} setActiveTab={(tab) => {
                if (tab === 'settings') setIsSettingsOpen(true);
                else setActiveTab(tab);
            }} collapsed={sidebarCollapsed} onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)} />

            {/* Main Layout (Left Margin for Sidebar) */}
            <div className={cn("flex-1 flex flex-col min-w-0 h-full transition-all duration-300", sidebarCollapsed ? "ml-16" : "ml-64")}>

                {/* Fixed Header */}
                <Header
                    activeTab={activeTab}
                    onAddDownload={() => setIsModalOpen(true)}
                    onPauseAll={() => AppBinding.PauseAllDownloads().catch(console.error)}
                    onResumeAll={() => AppBinding.ResumeAllDownloads().catch(console.error)}
                    sidebarCollapsed={sidebarCollapsed}
                />

                {/* Scrollable Content Area */}
                <main className="flex-1 min-h-0 pt-16 pb-7 bg-th-base flex flex-col">

                        {/* Dynamic Content */}
                        {activeTab === 'analytics' ? (
                            <div className="flex-1 overflow-y-auto scrollbar-thin scrollbar-thumb-th-raised scrollbar-track-transparent">
                                <div className="max-w-[1600px] mx-auto p-4 sm:p-6 md:p-8">
                                    <AnalyticsTab />
                                </div>
                            </div>
                        ) : activeTab === 'speedtest' ? (
                            <div className="flex-1 overflow-y-auto scrollbar-thin scrollbar-thumb-th-raised scrollbar-track-transparent">
                                <div className="max-w-[1600px] mx-auto p-4 sm:p-6 md:p-8">
                                    <SpeedTestTab />
                                </div>
                            </div>
                        ) : (
                            <div className="flex-1 min-h-0 flex flex-col px-5 pt-4 pb-3">
                                {/* Filter Bar: Search + Dropdowns */}
                                <div className="flex items-center gap-2 mb-2.5 shrink-0">
                                    <div className="relative flex-1">
                                        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-th-text-m" />
                                        <input
                                            type="text"
                                            placeholder="Search by filename or URL..."
                                            value={searchQuery}
                                            onChange={e => setSearchQuery(e.target.value)}
                                            className="w-full bg-th-surface border border-th-border rounded-lg pl-10 pr-4 py-1.5 text-sm text-th-text placeholder-th-text-m focus:border-th-accent focus:outline-none"
                                        />
                                    </div>
                                    <Dropdown
                                        value={statusFilter}
                                        onChange={v => setStatusFilter(v as StatusFilter)}
                                        options={[
                                            { value: 'all', label: 'All Status' },
                                            { value: 'downloading', label: 'Active' },
                                            { value: 'completed', label: 'Completed' },
                                            { value: 'paused', label: 'Paused' },
                                            { value: 'pending', label: 'Queued' },
                                            { value: 'error', label: 'Failed' },
                                        ]}
                                    />
                                    <Dropdown
                                        value={categoryFilter}
                                        onChange={v => setCategoryFilter(v as CategoryFilter)}
                                        options={[
                                            { value: 'all', label: 'All Types' },
                                            { value: 'video', label: 'Video' },
                                            { value: 'compressed', label: 'Archives' },
                                            { value: 'document', label: 'Documents' },
                                            { value: 'program', label: 'Programs' },
                                            { value: 'other', label: 'Other' },
                                        ]}
                                    />
                                </div>

                                {/* Data Grid - fills remaining height */}
                                <div className="flex-1 min-h-0 bg-th-surface border border-th-border rounded-xl overflow-hidden shadow-lg shadow-black/5 flex flex-col">
                                    <DownloadsTable
                                        data={filteredDownloads}
                                        onOpenFile={openFile}
                                        onOpenFolder={openFolder}
                                        onReorder={reorderDownload}
                                        onSetPriority={setPriority}
                                        addToast={addToast}
                                        selectedIds={selectedIds}
                                        onSelectionChange={setSelectedIds}
                                    />
                                </div>
                            </div>
                        )}
                </main>

                <StatusBar
                    activeDownloads={Object.values(downloads).filter((d: any) => d.status === 'downloading').length}
                    pendingDownloads={Object.values(downloads).filter((d: any) => d.status === 'pending').length}
                    dailyData={dailyData}
                    globalSpeed={totalSpeed}
                    sidebarCollapsed={sidebarCollapsed}
                />
            </div>

            <AddURLModal
                isOpen={isModalOpen}
                onClose={() => { setIsModalOpen(false); setPasteUrl(undefined); }}
                onAdd={addDownload}
                initialUrl={pasteUrl}
            />

            <SettingsModal
                isOpen={isSettingsOpen}
                onClose={() => setIsSettingsOpen(false)}
            />
        </div>
    );
}

export default App;
