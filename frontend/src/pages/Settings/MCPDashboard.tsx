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
            <div className={`p-6 rounded-xl border transition-colors ${enabled ? "bg-cyan-900/20 border-cyan-500/30" : "bg-gray-800/50 border-gray-700"}`}>
                <div className="flex items-start justify-between">
                    <div className="flex gap-4">
                        <div className={`p-3 rounded-lg h-fit ${enabled ? "bg-cyan-500/10 text-cyan-400" : "bg-gray-700/50 text-gray-500"}`}>
                            <Bot size={28} />
                        </div>
                        <div>
                            <h3 className="text-lg font-bold text-white mb-1">MCP Server & API</h3>
                            <p className="text-sm text-gray-400 max-w-sm">
                                Enable the Model Context Protocol (MCP) server and HTTP Control API to allow AI agents to manage downloads.
                            </p>
                            <div className="mt-4 grid grid-cols-2 gap-2 text-xs font-mono">
                                <div className="bg-black/30 p-2 rounded border border-gray-700">
                                    <span className="text-gray-500 block">Status</span>
                                    <span className={enabled ? "text-green-400" : "text-gray-500"}>
                                        {enabled ? "ACTIVE" : "DISABLED"}
                                    </span>
                                </div>
                                <div className="bg-black/30 p-2 rounded border border-gray-700">
                                    <span className="text-gray-500 block">Transport</span>
                                    <span className="text-cyan-400">STDIN + HTTP</span>
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
                        <div className="w-14 h-7 bg-gray-700 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-cyan-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[4px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-6 after:w-6 after:transition-all peer-checked:bg-cyan-600"></div>
                    </label>
                </div>
            </div>

            {/* Configuration */}
            {enabled && (
                <div className="bg-gray-900/50 border border-gray-800 rounded-xl p-6 space-y-6">
                    <h4 className="text-md font-medium text-white border-b border-gray-800 pb-2">Configuration</h4>

                    {/* Port & Concurrent */}
                    <div className="grid grid-cols-2 gap-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-400 mb-2">HTTP API Port</label>
                            <div className="relative">
                                <input
                                    type="number"
                                    value={port}
                                    onChange={handlePortChange}
                                    className="w-full bg-black/30 border border-gray-700 rounded-lg p-2.5 text-white focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                                />
                                {showRestartWarning && (
                                    <div className="absolute right-0 top-0 h-full flex items-center pr-3 pointer-events-none">
                                        <AlertTriangle className="text-yellow-500" size={16} />
                                    </div>
                                )}
                            </div>
                            <p className="text-xs text-gray-500 mt-1">Requires restart to apply changes.</p>
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-400 mb-2">Max Concurrent Requests</label>
                            <input
                                type="number"
                                value={maxConcurrent}
                                onChange={handleConcurrentChange}
                                className="w-full bg-black/30 border border-gray-700 rounded-lg p-2.5 text-white focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 outline-none"
                            />
                            <p className="text-xs text-gray-500 mt-1">Simultaneous AI tool calls processed.</p>
                        </div>
                    </div>

                    {/* Auth Token */}
                    <div>
                        <label className="block text-sm font-medium text-gray-400 mb-2">Access Token (X-Tachyon-Token)</label>
                        <div className="flex gap-2">
                            <code className="flex-1 bg-black/50 border border-gray-700 rounded-lg p-2.5 text-cyan-300 font-mono text-sm break-all">
                                {token}
                            </code>
                            <button onClick={copyToken} className="bg-gray-800 hover:bg-gray-700 text-white p-2.5 rounded-lg border border-gray-700 transition-colors">
                                <Copy size={18} />
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};
