import { useState } from 'react';
import { Plus } from 'lucide-react';
import { Sidebar } from './components/Sidebar';
import { DownloadItem } from './components/DownloadItem';
import { AddURLModal } from './components/AddURLModal';
import { useTachyon } from './hooks/useTachyon';

function App() {
    const [activeTab, setActiveTab] = useState("all");
    const [isModalOpen, setIsModalOpen] = useState(false);
    const { downloads, addDownload, openFolder } = useTachyon();

    // Filter downloads based on active tab
    const filteredDownloads = Object.values(downloads).filter(item => {
        if (activeTab === "all") return true;
        return item.status === activeTab;
    });

    return (
        <div className="flex h-screen bg-gray-950 text-white font-sans overflow-hidden select-none">
            <Sidebar activeTab={activeTab} setActiveTab={setActiveTab} />

            <main className="flex-1 flex flex-col min-w-0">
                {/* Header */}
                <header className="h-16 border-b border-gray-800 flex items-center justify-between px-8 bg-gray-950/50 backdrop-blur-sm z-10 dragging-header">
                    {/* Window Drag Area */}
                    <div className="flex items-center gap-4">
                        <h2 className="text-lg font-semibold capitalize text-gray-200">
                            {activeTab === "all" ? "All Downloads" : activeTab}
                        </h2>
                        <span className="px-2 py-0.5 bg-gray-800 text-gray-400 text-xs rounded-full">
                            {filteredDownloads.length}
                        </span>
                    </div>

                    <button
                        onClick={() => setIsModalOpen(true)}
                        className="flex items-center gap-2 bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded-lg text-sm font-medium transition-all shadow-lg shadow-blue-900/20 active:scale-95"
                    >
                        <Plus size={18} />
                        Add Download
                    </button>
                </header>

                {/* Content Area */}
                <div className="flex-1 overflow-auto p-8">
                    <div className="max-w-4xl mx-auto space-y-4">
                        {filteredDownloads.length === 0 ? (
                            <div className="text-center py-20 text-gray-500">
                                <p>No downloads found in this category.</p>
                            </div>
                        ) : (
                            filteredDownloads.map(item => (
                                <DownloadItem
                                    key={item.id}
                                    item={item}
                                    onOpenFolder={openFolder}
                                />
                            ))
                        )}
                    </div>
                </div>
            </main>

            <AddURLModal
                isOpen={isModalOpen}
                onClose={() => setIsModalOpen(false)}
                onAdd={addDownload}
            />
        </div>
    );
}

export default App;
