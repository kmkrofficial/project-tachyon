import React, { useEffect, useState } from 'react';
import { useSettingsStore } from '../../store';
import { Sparkles, AlertTriangle, Trash2 } from 'lucide-react';
// @ts-ignore
import { FactoryReset, GetEnableAI } from '../../../wailsjs/go/app/App';

export const GeneralSettings: React.FC = () => {
    const settings = useSettingsStore();
    const [enableAI, setEnableAI] = useState(false);
    const [showResetConfirm, setShowResetConfirm] = useState(false);

    useEffect(() => {
        // Load initial state
        if (GetEnableAI) {
            GetEnableAI().then(setEnableAI);
        }
    }, []);

    const handleFactoryReset = async () => {
        try {
            if (FactoryReset) {
                await FactoryReset();
                // Reload the window to reset state
                window.location.reload();
            }
        } catch (error) {
            console.error("Factory Reset Failed:", error);
            alert("Factory reset failed. Check logs.");
        }
    };

    return (
        <div className="space-y-8">
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

            <hr className="border-slate-800" />

            {/* Danger Zone */}
            <div className="space-y-4">
                <h3 className="text-red-500 font-medium flex items-center gap-2">
                    <AlertTriangle size={18} />
                    Danger Zone
                </h3>

                <div className="bg-slate-900/50 border border-red-500/20 rounded-lg p-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <h4 className="text-slate-200 font-medium">Factory Reset</h4>
                            <p className="text-sm text-slate-500 mt-1">
                                Wipes all download history, settings, and local data. This action cannot be undone.
                            </p>
                        </div>

                        {!showResetConfirm ? (
                            <button
                                onClick={() => setShowResetConfirm(true)}
                                className="px-4 py-2 bg-red-500/10 hover:bg-red-500/20 text-red-500 border border-red-500/50 rounded-lg transition-colors flex items-center gap-2 font-medium text-sm"
                            >
                                <Trash2 size={16} />
                                Reset Everything
                            </button>
                        ) : (
                            <div className="flex items-center gap-2 animate-in fade-in slide-in-from-right-4 duration-300">
                                <span className="text-sm text-red-400 font-medium mr-2">Are you sure?</span>
                                <button
                                    onClick={handleFactoryReset}
                                    className="px-3 py-1.5 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors text-sm font-bold shadow-lg shadow-red-900/20"
                                >
                                    Yes, Wipe It
                                </button>
                                <button
                                    onClick={() => setShowResetConfirm(false)}
                                    className="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded-lg transition-colors text-sm"
                                >
                                    Cancel
                                </button>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
};
