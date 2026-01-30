import React, { useEffect, useState } from 'react';
import { useSettingsStore } from '../../store';
import { Sparkles } from 'lucide-react';

export const GeneralSettings: React.FC = () => {
    const settings = useSettingsStore();
    const [enableAI, setEnableAI] = useState(false);

    useEffect(() => {
        // Load initial state
        if (window.go?.main?.App?.GetEnableAI) {
            window.go.main.App.GetEnableAI().then(setEnableAI);
        }
    }, []);

    const toggleAI = (checked: boolean) => {
        setEnableAI(checked);
        if (window.go?.main?.App?.SetEnableAI) {
            window.go.main.App.SetEnableAI(checked);
        }
    };

    return (
        <div className="space-y-8">
            {/* AI Section */}
            <div className="bg-gradient-to-r from-purple-900/20 to-blue-900/20 border border-purple-500/30 rounded-xl p-6">
                <div className="flex items-start justify-between">
                    <div className="flex gap-4">
                        <div className="p-3 bg-purple-500/10 rounded-lg h-fit">
                            <Sparkles className="text-purple-400" size={24} />
                        </div>
                        <div>
                            <h3 className="text-lg font-bold text-white mb-1">AI Interface</h3>
                            <p className="text-sm text-purple-200/60 max-w-sm">
                                Enable the external MCP Server and HTTP API to allow AI Agents (like me!) to control Tachyon.
                            </p>
                            <div className="mt-4 flex flex-col gap-2 text-xs text-gray-500 font-mono bg-black/40 p-3 rounded border border-purple-500/20">
                                <div><span className="text-purple-500">MCP:</span> Stdin/Stdout (JSON-RPC)</div>
                                <div><span className="text-blue-500">API:</span> http://127.0.0.1:4444</div>
                            </div>
                        </div>
                    </div>

                    <label className="relative inline-flex items-center cursor-pointer">
                        <input
                            type="checkbox"
                            className="sr-only peer"
                            checked={enableAI}
                            onChange={(e) => toggleAI(e.target.checked)}
                        />
                        <div className="w-14 h-7 bg-gray-700 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-purple-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-0.5 after:left-[4px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-6 after:w-6 after:transition-all peer-checked:bg-purple-600"></div>
                    </label>
                </div>
            </div>

            {/* Standard Settings */}
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

            <div className="flex items-center justify-between opacity-50 pointer-events-none">
                <span className="text-gray-300">File Categorization (Always On)</span>
                <div className="w-10 h-5 bg-blue-600 rounded-full relative">
                    <div className="absolute right-1 top-1 w-3 h-3 bg-white rounded-full"></div>
                </div>
            </div>
        </div>
    );
};
