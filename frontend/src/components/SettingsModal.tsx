import React, { useState } from 'react';
import { X, Save, Clock, Wifi, Sliders, Shield } from 'lucide-react';
import { useSettingsStore } from '../store';
import { useTachyon } from '../hooks/useTachyon';

import { GeneralSettings } from '../pages/Settings/General';
import { SecurityLog } from '../pages/Settings/SecurityLog';

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose }) => {
    const settings = useSettingsStore();
    const [activeTab, setActiveTab] = useState<'general' | 'scheduler' | 'network' | 'security'>('general');

    if (!isOpen) return null;

    const handleSave = () => {
        // Apply settings to backend
        if (window.go?.main?.App?.SetMaxConcurrentDownloads) {
            window.go.main.App.SetMaxConcurrentDownloads(settings.maxConcurrentDownloads);
        }
        if (window.go?.main?.App?.SetThreadsPerDownload) {
            // Assuming we have a backend method for this, otherwise it's just local
            // window.go.main.App.SetThreadsPerDownload(settings.threadsPerDownload);
        }
        onClose();
    };

    // ...

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
                        <TabButton
                            id="security"
                            label="Security"
                            icon={Shield}
                            active={activeTab === 'security'}
                            onClick={() => setActiveTab('security')}
                        />
                    </div>

                    {/* Content */}
                    <div className="flex-1 p-6 overflow-y-auto">
                        {activeTab === 'general' && <GeneralSettings />}

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
                                <div className="bg-gray-800 p-4 rounded-lg">
                                    <span className="block text-white font-medium mb-4">Concurrency Limits per Host</span>
                                    <p className="text-xs text-gray-400 mb-4">Limit simultaneous downloads from specific websites (e.g. mega.nz to 1).</p>

                                    <div className="flex gap-2 mb-4">
                                        <input type="text" placeholder="example.com" className="bg-black/30 border border-gray-600 rounded p-2 text-gray-300 text-sm flex-1" id="hostInput" />
                                        <input type="number" min="1" max="10" placeholder="1" className="bg-black/30 border border-gray-600 rounded p-2 text-gray-300 text-sm w-16" id="limitInput" />
                                        <button
                                            className="px-3 bg-blue-600 hover:bg-blue-500 text-white rounded text-sm font-medium"
                                            onClick={() => {
                                                const host = (document.getElementById('hostInput') as HTMLInputElement).value;
                                                const limit = parseInt((document.getElementById('limitInput') as HTMLInputElement).value);
                                                if (host && limit > 0) {
                                                    // @ts-ignore
                                                    if (window.go?.main?.App?.SetHostLimit) {
                                                        // @ts-ignore
                                                        window.go.main.App.SetHostLimit(host, limit);
                                                    }
                                                }
                                            }}
                                        >Add Rule</button>
                                    </div>
                                </div>
                            </div>
                        )}

                        {activeTab === 'security' && <SecurityLog />}
                    </div>
                </div >

                <div className="p-4 border-t border-gray-800 flex justify-end bg-gray-900/50">
                    <button
                        onClick={handleSave}
                        className="flex items-center gap-2 bg-blue-600 hover:bg-blue-500 text-white px-6 py-2 rounded-lg font-medium transition-colors"
                    >
                        <Save size={18} /> Save Changes
                    </button>
                </div>
            </div >
        </div >
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
