import React, { useState } from 'react';
import { X, Save, Clock, Wifi, Sliders } from 'lucide-react';
import { useSettingsStore } from '../store';
import { useTachyon } from '../hooks/useTachyon';

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose }) => {
    const settings = useSettingsStore();
    const { runSpeedTest } = useTachyon();
    const [activeTab, setActiveTab] = useState<'general' | 'scheduler' | 'network'>('general');
    const [speedResult, setSpeedResult] = useState<string | null>(null);

    if (!isOpen) return null;

    const handleSave = () => {
        onClose();
    };

    const handleSpeedTest = async () => {
        setSpeedResult("Testing...");
        try {
            const res = await runSpeedTest();
            if (res) {
                setSpeedResult(`${res.download_mbps.toFixed(1)} Mbps`);
            } else {
                setSpeedResult("Failed");
            }
        } catch {
            setSpeedResult("Error");
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
            <div className="bg-gray-900 w-full max-w-2xl rounded-2xl border border-gray-800 shadow-2xl overflow-hidden flex flex-col max-h-[80vh]">

                {/* Header */}
                <div className="flex justify-between items-center p-6 border-b border-gray-800 bg-gray-900/50">
                    <h2 className="text-xl font-bold text-white">Settings</h2>
                    <button onClick={onClose} className="p-1 hover:bg-gray-800 rounded-full text-gray-400 hover:text-white">
                        <X size={20} />
                    </button>
                </div>

                <div className="flex flex-1 overflow-hidden">
                    {/* Sidebar Tabs */}
                    <div className="w-48 bg-gray-950 border-r border-gray-800 p-4 space-y-2">
                        <TabButton
                            id="general"
                            label="General"
                            icon={Sliders}
                            active={activeTab === 'general'}
                            onClick={() => setActiveTab('general')}
                        />
                        <TabButton
                            id="scheduler"
                            label="Scheduler"
                            icon={Clock}
                            active={activeTab === 'scheduler'}
                            onClick={() => setActiveTab('scheduler')}
                        />
                        <TabButton
                            id="network"
                            label="Network"
                            icon={Wifi}
                            active={activeTab === 'network'}
                            onClick={() => setActiveTab('network')}
                        />
                    </div>

                    {/* Content */}
                    <div className="flex-1 p-6 overflow-y-auto">
                        {activeTab === 'general' && (
                            <div className="space-y-6">
                                <div>
                                    <label className="block text-sm font-medium text-gray-400 mb-2">Max Concurrent Downloads: {settings.maxConcurrentDownloads}</label>
                                    <input
                                        type="range" min="1" max="10"
                                        value={settings.maxConcurrentDownloads}
                                        onChange={(e) => settings.setMaxConcurrentDownloads(parseInt(e.target.value))}
                                        className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-400 mb-2">Threads per Download: {settings.threadsPerDownload}</label>
                                    <input
                                        type="range" min="1" max="32"
                                        value={settings.threadsPerDownload}
                                        onChange={(e) => settings.setThreadsPerDownload(parseInt(e.target.value))}
                                        className="w-full h-2 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-purple-500"
                                    />
                                </div>
                                <div className="flex items-center justify-between">
                                    <span className="text-gray-300">File Categorization</span>
                                    <div className="w-10 h-5 bg-blue-600 rounded-full cursor-pointer relative">
                                        <div className="absolute right-1 top-1 w-3 h-3 bg-white rounded-full"></div>
                                    </div>
                                </div>
                            </div>
                        )}

                        {activeTab === 'scheduler' && (
                            <div className="space-y-6">
                                <div className="bg-gray-800/50 p-4 rounded-lg border border-gray-700">
                                    <p className="text-sm text-yellow-400 mb-2">Schedule Activity</p>
                                    <div className="grid grid-cols-2 gap-4">
                                        <div>
                                            <label className="block text-xs text-gray-500 mb-1">Start Time</label>
                                            <input type="time" className="bg-gray-900 border border-gray-700 rounded p-2 text-white w-full" defaultValue="02:00" />
                                        </div>
                                        <div>
                                            <label className="block text-xs text-gray-500 mb-1">Stop Time</label>
                                            <input type="time" className="bg-gray-900 border border-gray-700 rounded p-2 text-white w-full" defaultValue="08:00" />
                                        </div>
                                    </div>
                                </div>
                            </div>
                        )}

                        {activeTab === 'network' && (
                            <div className="space-y-6">
                                <div className="flex justify-between items-center bg-gray-800 p-4 rounded-lg">
                                    <div>
                                        <span className="block text-white font-medium">Speed Test</span>
                                        <span className="text-xs text-gray-500">Check connection to servers</span>
                                    </div>
                                    <div className="flex items-center gap-4">
                                        {speedResult && <span className="text-green-400 font-mono text-sm">{speedResult}</span>}
                                        <button onClick={handleSpeedTest} className="px-3 py-1 bg-blue-600 rounded text-sm hover:bg-blue-500">Run</button>
                                    </div>
                                </div>
                                <div className="bg-gray-800 p-4 rounded-lg">
                                    <span className="block text-white font-medium mb-2">Remote Access</span>
                                    <div className="flex gap-2">
                                        <input readOnly value="192.168.1.45" className="bg-black/30 border border-gray-600 rounded p-1 text-gray-400 text-xs flex-1" />
                                        <button className="text-blue-400 text-xs hover:text-white">Copy JSON</button>
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                </div>

                <div className="p-4 border-t border-gray-800 flex justify-end bg-gray-900/50">
                    <button
                        onClick={handleSave}
                        className="flex items-center gap-2 bg-blue-600 hover:bg-blue-500 text-white px-6 py-2 rounded-lg font-medium transition-colors"
                    >
                        <Save size={18} /> Save Changes
                    </button>
                </div>
            </div>
        </div>
    );
};

const TabButton = ({ id, label, icon: Icon, active, onClick }: any) => (
    <button
        onClick={onClick}
        className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${active ? "bg-gray-800 text-white" : "text-gray-400 hover:text-gray-200"}`}
    >
        <Icon size={16} />
        {label}
    </button>
);
