import React, { useState } from 'react';
import { X, Save, Wifi, Sliders, Bot } from 'lucide-react';
import { useSettingsStore } from '../store';
import { useTachyon } from '../hooks/useTachyon';

import { GeneralSettings } from '../pages/Settings/General';
import { MCPDashboard } from '../pages/Settings/MCPDashboard';
import { NetworkSettings } from '../pages/Settings/NetworkSettings';

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose }) => {
    const settings = useSettingsStore();
    const [activeTab, setActiveTab] = useState<'general' | 'network' | 'mcp'>('general');

    if (!isOpen) return null;

    const handleSave = () => {
        // ... (rest of handleSave)
        if (window.go?.app?.App?.SetMaxConcurrentDownloads) {
            window.go.app.App.SetMaxConcurrentDownloads(settings.maxConcurrentDownloads);
        }
        if (window.go?.app?.App?.SetGlobalSpeedLimit) {
            window.go.app.App.SetGlobalSpeedLimit(settings.globalSpeedLimit);
        }
        onClose();
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
            <div className="bg-th-surface w-full max-w-2xl rounded-2xl border border-th-border shadow-2xl overflow-hidden flex flex-col h-[80vh]">

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
                    </div>

                    {/* Content */}
                    <div className="flex-1 p-6 overflow-y-auto">
                        {activeTab === 'general' && <GeneralSettings />}
                        {activeTab === 'mcp' && <MCPDashboard />}
                        {activeTab === 'network' && <NetworkSettings />}
                    </div>
                </div >

                <div className="p-4 border-t border-th-border flex justify-end bg-th-surface/50">
                    <button
                        onClick={handleSave}
                        className="flex items-center gap-2 bg-th-accent hover:bg-th-accent-h text-white px-6 py-2 rounded-lg font-medium transition-colors"
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
