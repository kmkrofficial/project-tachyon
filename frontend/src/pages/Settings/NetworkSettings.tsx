import React, { useState, useEffect } from 'react';
import { useSettingsStore } from '../../store';

type SpeedUnit = 'KBps' | 'MBps' | 'GBps';

const unitMultipliers: Record<SpeedUnit, number> = {
    KBps: 1024,
    MBps: 1024 * 1024,
    GBps: 1024 * 1024 * 1024,
};

const bytesToUnit = (bytes: number): { value: string; unit: SpeedUnit } => {
    if (bytes === 0) return { value: '', unit: 'MBps' };
    if (bytes >= 1024 * 1024 * 1024) return { value: String(Math.round((bytes / (1024 * 1024 * 1024)) * 100) / 100), unit: 'GBps' };
    if (bytes >= 1024 * 1024) return { value: String(Math.round((bytes / (1024 * 1024)) * 100) / 100), unit: 'MBps' };
    return { value: String(Math.round((bytes / 1024) * 100) / 100), unit: 'KBps' };
};

export const NetworkSettings: React.FC = () => {
    const settings = useSettingsStore();
    const initial = bytesToUnit(settings.globalSpeedLimit);
    const [speedValue, setSpeedValue] = useState(initial.value);
    const [speedUnit, setSpeedUnit] = useState<SpeedUnit>(initial.unit);

    useEffect(() => {
        const num = parseFloat(speedValue);
        if (!speedValue || isNaN(num) || num <= 0) {
            settings.setGlobalSpeedLimit(0);
        } else {
            settings.setGlobalSpeedLimit(Math.round(num * unitMultipliers[speedUnit]));
        }
    }, [speedValue, speedUnit]);

    return (
        <div className="space-y-6">
            <div>
                <label className="block text-sm font-medium text-th-text mb-2">
                    Concurrent Downloads: {settings.maxConcurrentDownloads === 0 ? 'Unlimited' : settings.maxConcurrentDownloads}
                </label>
                <input
                    type="range" min="0" max="10"
                    value={settings.maxConcurrentDownloads}
                    onChange={(e) => settings.setMaxConcurrentDownloads(parseInt(e.target.value))}
                    className="w-full h-2 bg-th-overlay rounded-lg appearance-none cursor-pointer accent-th-accent"
                />
                <div className="flex justify-between text-xs text-th-text-m mt-1">
                    <span>Unlimited</span><span>10</span>
                </div>
            </div>

            <div>
                <label className="block text-sm font-medium text-th-text mb-2">
                    Threads per Download: {settings.threadsPerDownload}
                </label>
                <input
                    type="range" min="4" max="32" step="4"
                    value={settings.threadsPerDownload}
                    onChange={(e) => settings.setThreadsPerDownload(parseInt(e.target.value))}
                    className="w-full h-2 bg-th-overlay rounded-lg appearance-none cursor-pointer accent-purple-500"
                />
                <div className="flex justify-between text-xs text-th-text-m mt-1">
                    <span>4</span><span>8</span><span>12</span><span>16</span><span>20</span><span>24</span><span>28</span><span>32</span>
                </div>
            </div>

            <div>
                <label className="block text-sm font-medium text-th-text mb-2">
                    Download Retries: {settings.downloadRetries === 0 ? 'Disabled' : settings.downloadRetries}
                </label>
                <input
                    type="range" min="0" max="10"
                    value={settings.downloadRetries}
                    onChange={(e) => settings.setDownloadRetries(parseInt(e.target.value))}
                    className="w-full h-2 bg-th-overlay rounded-lg appearance-none cursor-pointer accent-green-500"
                />
                <div className="flex justify-between text-xs text-th-text-m mt-1">
                    <span>Off</span><span>10</span>
                </div>
            </div>

            {/* Global Speed Limit */}
            <div>
                <label className="block text-sm font-medium text-th-text mb-2">
                    Global Speed Limit
                </label>
                <div className="flex items-center gap-2">
                    <input
                        type="text"
                        inputMode="decimal"
                        placeholder="Unlimited"
                        value={speedValue}
                        onChange={e => setSpeedValue(e.target.value.replace(/[^0-9.]/g, ''))}
                        className="flex-1 bg-th-surface border border-th-border rounded-lg px-3 py-2 text-sm text-th-text placeholder-th-text-m focus:border-th-accent focus:outline-none"
                    />
                    <select
                        value={speedUnit}
                        onChange={e => setSpeedUnit(e.target.value as SpeedUnit)}
                        className="bg-th-surface border border-th-border rounded-lg px-3 py-2 text-sm text-th-text focus:border-th-accent focus:outline-none"
                    >
                        <option value="KBps">KB/s</option>
                        <option value="MBps">MB/s</option>
                        <option value="GBps">GB/s</option>
                    </select>
                </div>
                <p className="text-xs text-th-text-m mt-1.5">Leave empty for unlimited speed.</p>
            </div>
        </div>
    );
};
