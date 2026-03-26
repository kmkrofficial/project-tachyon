import React, { useEffect, useState } from 'react';
import { useSettingsStore } from '../../store';
import { Bot, Copy, RefreshCw, AlertTriangle } from 'lucide-react';

export const MCPDashboard: React.FC = () => {
    const [enabled, setEnabled] = useState(false);
    const [port, setPort] = useState(4444);
    const [maxConcurrent, setMaxConcurrent] = useState(5);
    const [token, setToken] = useState("loading...");
    const [showRestartWarning, setShowRestartWarning] = useState(false);

    useEffect(() => {
        if (window.go?.app?.App) {
            window.go.app.App.GetEnableAI().then(setEnabled);
            window.go.app.App.GetAIPort().then(setPort);
            window.go.app.App.GetAIMaxConcurrent().then(setMaxConcurrent);
            window.go.app.App.GetAIToken().then(setToken);
        }
    }, []);

    const handleEnableChange = (checked: boolean) => {
        setEnabled(checked);
        window.go?.app?.App?.SetEnableAI(checked);
    };

    const handlePortChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const val = parseInt(e.target.value);
        setPort(val);
        window.go?.app?.App?.SetAIPort(val);
        setShowRestartWarning(true);
    };

    const handleConcurrentChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const val = parseInt(e.target.value);
        setMaxConcurrent(val);
        window.go?.app?.App?.SetAIMaxConcurrent(val);
    };

    const copyToken = () => {
        navigator.clipboard.writeText(token);
    };

    return (
        <div className="space-y-6">
            {/* Header / Main Toggle */}
            <div className={`p-6 rounded-xl border transition-colors ${enabled ? "bg-th-accent/10 border-th-accent/30" : "bg-th-raised/50 border-th-border"}`}>
                <div className="flex items-start justify-between">
                    <div className="flex gap-4">
                        <div className={`p-3 rounded-lg h-fit ${enabled ? "bg-th-accent/10 text-th-accent-t" : "bg-th-overlay/50 text-th-text-m"}`}>
                            <Bot size={28} />
                        </div>
                        <div>
                            <h3 className="text-lg font-bold text-th-text mb-1">MCP Server & API</h3>
                            <p className="text-sm text-th-text-s max-w-sm">
                                Enable the Model Context Protocol (MCP) server and HTTP Control API to allow AI agents to manage downloads.
                            </p>
                            <div className="mt-4 grid grid-cols-2 gap-2 text-xs font-mono">
                                <div className="bg-th-base/30 p-2 rounded border border-th-border">
                                    <span className="text-th-text-m block">Status</span>
                                    <span className={enabled ? "text-green-400" : "text-th-text-m"}>
                                        {enabled ? "ACTIVE" : "DISABLED"}
                                    </span>
                                </div>
                                <div className="bg-th-base/30 p-2 rounded border border-th-border">
                                    <span className="text-th-text-m block">Transport</span>
                                    <span className="text-th-accent-t">STDIN + HTTP</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <label className="relative inline-flex items-center cursor-pointer">
                        <input
                            type="checkbox"
                            className="sr-only peer"
                            checked={enabled}
                            onChange={(e) => handleEnableChange(e.target.checked)}
                        />
                        <div className="w-14 h-7 bg-th-overlay peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-th-accent/30 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[4px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-6 after:w-6 after:transition-all peer-checked:bg-th-accent"></div>
                    </label>
                </div>
            </div>

            {/* Configuration */}
            {enabled && (
                <div className="bg-th-surface/50 border border-th-border rounded-xl p-6 space-y-6">
                    <h4 className="text-md font-medium text-th-text border-b border-th-border pb-2">Configuration</h4>

                    {/* Port & Concurrent */}
                    <div className="grid grid-cols-2 gap-6">
                        <div>
                            <label className="block text-sm font-medium text-th-text-s mb-2">HTTP API Port</label>
                            <div className="relative">
                                <input
                                    type="number"
                                    value={port}
                                    onChange={handlePortChange}
                                    className="w-full bg-th-base/30 border border-th-border rounded-lg p-2.5 text-th-text focus:border-th-accent focus:ring-1 focus:ring-th-accent outline-none"
                                />
                                {showRestartWarning && (
                                    <div className="absolute right-0 top-0 h-full flex items-center pr-3 pointer-events-none">
                                        <AlertTriangle className="text-yellow-500" size={16} />
                                    </div>
                                )}
                            </div>
                            <p className="text-xs text-th-text-m mt-1">Requires restart to apply changes.</p>
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-th-text-s mb-2">Max Concurrent Requests</label>
                            <input
                                type="number"
                                value={maxConcurrent}
                                onChange={handleConcurrentChange}
                                className="w-full bg-th-base/30 border border-th-border rounded-lg p-2.5 text-th-text focus:border-th-accent focus:ring-1 focus:ring-th-accent outline-none"
                            />
                            <p className="text-xs text-th-text-m mt-1">Simultaneous AI tool calls processed.</p>
                        </div>
                    </div>

                    {/* Auth Token */}
                    <div>
                        <label className="block text-sm font-medium text-th-text-s mb-2">Access Token (X-Tachyon-Token)</label>
                        <div className="flex gap-2">
                            <code className="flex-1 bg-th-base/50 border border-th-border rounded-lg p-2.5 text-th-accent-t font-mono text-sm break-all">
                                {token}
                            </code>
                            <button onClick={copyToken} className="bg-th-raised hover:bg-th-overlay text-th-text p-2.5 rounded-lg border border-th-border transition-colors">
                                <Copy size={18} />
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};
