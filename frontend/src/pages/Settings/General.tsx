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
        </div>
    );
};
