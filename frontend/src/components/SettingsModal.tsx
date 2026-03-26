import React, { useState } from 'react';
import { X, Save, Clock, Wifi, Sliders, Shield, Bot } from 'lucide-react';
import { useSettingsStore } from '../store';
import { useTachyon } from '../hooks/useTachyon';

import { GeneralSettings } from '../pages/Settings/General';
import { SecurityLog } from '../pages/Settings/SecurityLog';
import { MCPDashboard } from '../pages/Settings/MCPDashboard';
import { NetworkSettings } from '../pages/Settings/NetworkSettings';

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose }) => {
    const settings = useSettingsStore();
    const [activeTab, setActiveTab] = useState<'general' | 'scheduler' | 'network' | 'security' | 'mcp'>('general');

    if (!isOpen) return null;

    const handleSave = () => {
        // ... (rest of handleSave)
        if (window.go?.app?.App?.SetMaxConcurrentDownloads) {
            window.go.app.App.SetMaxConcurrentDownloads(settings.maxConcurrentDownloads);
        }
        // ...
        onClose();
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
            <div className="bg-th-surface w-full max-w-2xl rounded-2xl border border-th-border shadow-2xl overflow-hidden flex flex-col max-h-[80vh]">

                {/* Header */}
                <div className="flex justify-between items-center p-6 border-b border-th-border bg-th-surface/50">
                    <h2 className="text-xl font-bold text-th-text">Settings</h2>
                    <button onClick={onClose} className="p-1 hover:bg-th-raised rounded-full text-th-text-s hover:text-th-text">
                        <X size={20} />
                    </button>
                </div>

                <div className="flex flex-1 overflow-hidden">
                    {/* Sidebar Tabs */}
                    <div className="w-48 bg-th-base border-r border-th-border p-4 space-y-2">
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
                            id="mcp"
                            label="MCP Server"
                            icon={Bot}
                            active={activeTab === 'mcp'}
                            onClick={() => setActiveTab('mcp')}
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
                        {activeTab === 'mcp' && <MCPDashboard />}

                        {activeTab === 'scheduler' && (
                            <div className="space-y-6">
                                <div className="bg-th-raised/50 p-4 rounded-lg border border-th-border-s">
                                    <p className="text-sm text-yellow-500 mb-2">Schedule Activity</p>
                                    <div className="grid grid-cols-2 gap-4">
                                        <div>
                                            <label className="block text-xs text-th-text-m mb-1">Start Time</label>
                                            <input type="time" className="bg-th-surface border border-th-border-s rounded p-2 text-th-text w-full" defaultValue="02:00" />
                                        </div>
                                        <div>
                                            <label className="block text-xs text-th-text-m mb-1">Stop Time</label>
                                            <input type="time" className="bg-th-surface border border-th-border-s rounded p-2 text-th-text w-full" defaultValue="08:00" />
                                        </div>
                                    </div>
                                </div>
                            </div>
                        )}

                        {activeTab === 'network' && <NetworkSettings />}

                        {activeTab === 'security' && <SecurityLog />}
                    </div>
                </div >

                <div className="p-4 border-t border-th-border flex justify-end bg-th-surface/50">
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
        className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${active ? "bg-th-raised text-th-text" : "text-th-text-s hover:text-th-text"}`}
    >
        <Icon size={16} />
        {label}
    </button>
);
