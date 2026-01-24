import { useState } from 'react';
import { Sidebar } from './components/Sidebar';
import { Header } from './components/Header';
import { DownloadsTable } from './components/DownloadsTable';
import { AddURLModal } from './components/AddURLModal';
import { SettingsModal } from './components/SettingsModal';
import { AnalyticsTab } from './components/AnalyticsTab';
import { useTachyon } from './hooks/useTachyon';

function App() {
    const [activeTab, setActiveTab] = useState("all");
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);

    // openFolder and openFile trigger backend ops by ID
    const { downloads, addDownload, openFolder, openFile } = useTachyon();

    // Filter downloads based on active tab
    const filteredDownloads = Object.values(downloads)
        .filter(item => {
            if (activeTab === "all") return true;
            if (activeTab === "settings" || activeTab === "analytics") return true; // Handled by render
            return item.status === activeTab;
        })
        .sort((a, b) => {
            // Sort by Created At Descending
            if (a.created_at && b.created_at) return b.created_at.localeCompare(a.created_at);
            return 0;
        });

    return (
        <div className="flex h-screen bg-slate-950 text-slate-200 font-sans overflow-hidden select-none">
            {/* Sidebar (Fixed Width) */}
            <Sidebar activeTab={activeTab} setActiveTab={(tab) => {
                if (tab === 'settings') setIsSettingsOpen(true);
                else setActiveTab(tab);
            }} />

            {/* Main Layout (Left Margin for Sidebar) */}
            <div className="flex-1 flex flex-col ml-64 min-w-0 h-full">

                {/* Fixed Header */}
                <Header onAddDownload={() => setIsModalOpen(true)} />

                {/* Scrollable Content Area */}
                <main className="flex-1 overflow-y-auto pt-16 bg-slate-950 scrollbar-thin scrollbar-thumb-slate-800 scrollbar-track-transparent">
                    <div className="max-w-[1600px] mx-auto p-6 md:p-8">

                        {/* Dynamic Content */}
                        {activeTab === 'analytics' ? (
                            <AnalyticsTab />
                        ) : (
                            <div className="space-y-6">
                                {/* Dashboard Widgets (Only on Dashboard) */}
                                {activeTab === 'all' && <DashboardWidgets downloads={Object.values(downloads)} />}

                                {/* Data Grid */}
                                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-2xl shadow-black/20">
                                    <DownloadsTable
                                        data={filteredDownloads}
                                        onOpenFile={openFile}
                                        onOpenFolder={openFolder}
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
const DashboardWidgets = ({ downloads }: { downloads: any[] }) => {
    const active = downloads.filter(d => d.status === 'downloading').length;
    const pending = downloads.filter(d => d.status === 'pending').length;
    // Mock speed calculation from active downloads if we had speed per item clearly aggregated
    const totalSpeed = downloads.reduce((acc, d) => acc + (d.speed_MBs || 0), 0);

    return (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <Widget title="Active Downloads" value={active.toString()} subtitle={`${totalSpeed.toFixed(1)} MB/s Total`} color="cyan" />
            <Widget title="Queue Pending" value={pending.toString()} subtitle="Next: 02:00 AM" color="indigo" />
            <Widget title="Data Today" value="12.4 GB" subtitle="Daily Usage" color="purple" />
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
