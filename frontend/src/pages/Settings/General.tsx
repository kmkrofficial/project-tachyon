import React, { useState, useEffect } from 'react';
import { useSettingsStore } from '../../store';
import { Sun, Moon, Monitor, FolderOpen, Eclipse } from 'lucide-react';

const Toggle: React.FC<{ enabled: boolean; onChange: (v: boolean) => void }> = ({ enabled, onChange }) => (
    <button
        onClick={() => onChange(!enabled)}
        className={`w-10 h-5 rounded-full relative transition-colors ${enabled ? 'bg-th-accent' : 'bg-th-overlay'}`}
    >
        <div className={`absolute top-1 w-3 h-3 bg-white rounded-full transition-all ${enabled ? 'right-1' : 'left-1'}`} />
    </button>
);

export const GeneralSettings: React.FC = () => {
    const settings = useSettingsStore();
    const [defaultPath, setDefaultPath] = useState('');

    useEffect(() => {
        // @ts-ignore
        window.go?.app?.App?.GetDefaultDownloadPath?.().then((p: string) => {
            if (p) setDefaultPath(p);
        });
    }, []);

    return (
        <div className="space-y-8">
            {/* Appearance */}
            <section>
                <h3 className="text-sm font-semibold text-th-text-s uppercase tracking-wider mb-4">Appearance</h3>
                <div className="space-y-5">
                    <div>
                        <label className="block text-sm font-medium text-th-text mb-2">Theme</label>
                        <div className="flex gap-2">
                            {([
                                { value: 'dark' as const, label: 'Dark', Icon: Moon },
                                { value: 'black' as const, label: 'Black', Icon: Eclipse },
                                { value: 'light' as const, label: 'Light', Icon: Sun },
                                { value: 'system' as const, label: 'System', Icon: Monitor },
                            ]).map(({ value, label, Icon }) => (
                                <button
                                    key={value}
                                    onClick={() => settings.setTheme(value)}
                                    className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors border ${
                                        settings.theme === value
                                            ? 'bg-th-accent/20 border-th-accent text-th-accent-t'
                                            : 'bg-th-raised border-th-border text-th-text-s hover:text-th-text'
                                    }`}
                                >
                                    <Icon size={16} />
                                    {label}
                                </button>
                            ))}
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-th-text mb-2">Time Display</label>
                        <div className="flex gap-2">
                            {([
                                { value: 'relative' as const, label: 'Relative', tooltip: 'e.g. "2 minutes ago"' },
                                { value: 'absolute' as const, label: 'Absolute', tooltip: 'e.g. "14:32:05"' },
                            ]).map(({ value, label, tooltip }) => (
                                <button
                                    key={value}
                                    title={tooltip}
                                    onClick={() => settings.setTimeFormat(value)}
                                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors border ${
                                        settings.timeFormat === value
                                            ? 'bg-th-accent/20 border-th-accent text-th-accent-t'
                                            : 'bg-th-raised border-th-border text-th-text-s hover:text-th-text'
                                    }`}
                                >
                                    {label}
                                </button>
                            ))}
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-th-text mb-2">Size Unit</label>
                        <div className="flex gap-2">
                            {(['auto', 'KB', 'MB', 'GB'] as const).map((unit) => (
                                <button
                                    key={unit}
                                    onClick={() => settings.setSizeUnit(unit)}
                                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors border ${
                                        settings.sizeUnit === unit
                                            ? 'bg-th-accent/20 border-th-accent text-th-accent-t'
                                            : 'bg-th-raised border-th-border text-th-text-s hover:text-th-text'
                                    }`}
                                >
                                    {unit === 'auto' ? 'Auto' : unit}
                                </button>
                            ))}
                        </div>
                    </div>
                </div>
            </section>

            <hr className="border-th-border" />

            {/* Behavior */}
            <section>
                <h3 className="text-sm font-semibold text-th-text-s uppercase tracking-wider mb-4">Behavior</h3>
                <div className="space-y-5">
                    <div>
                        <label className="block text-sm font-medium text-th-text mb-2">
                            <FolderOpen size={14} className="inline mr-1.5 -mt-0.5" />
                            Default Download Path
                        </label>
                        <input
                            type="text"
                            placeholder={defaultPath}
                            value={settings.downloadPath}
                            onChange={(e) => settings.setDownloadPath(e.target.value)}
                            className="w-full bg-th-surface border border-th-border rounded-lg px-3 py-2 text-sm text-th-text placeholder-th-text-m font-mono focus:border-th-accent focus:outline-none"
                        />
                    </div>

                    <div className="flex items-center justify-between">
                        <div>
                            <span className="text-sm font-medium text-th-text">Start on Boot</span>
                            <p className="text-xs text-th-text-m mt-0.5">Launch TDM when your system starts</p>
                        </div>
                        <Toggle enabled={settings.startOnBoot} onChange={settings.setStartOnBoot} />
                    </div>

                    <div className="flex items-center justify-between">
                        <div>
                            <span className="text-sm font-medium text-th-text">Close to System Tray</span>
                            <p className="text-xs text-th-text-m mt-0.5">Minimize to tray instead of quitting</p>
                        </div>
                        <Toggle enabled={settings.closeToTray} onChange={settings.setCloseToTray} />
                    </div>

                    <div className="flex items-center justify-between">
                        <div>
                            <span className="text-sm font-medium text-th-text">File Categorization</span>
                            <p className="text-xs text-th-text-m mt-0.5">Auto-sort files into folders by type</p>
                        </div>
                        <Toggle enabled={settings.fileCategorization} onChange={settings.setFileCategorization} />
                    </div>

                    <div className="flex items-center justify-between">
                        <div>
                            <span className="text-sm font-medium text-th-text">Quick Download</span>
                            <p className="text-xs text-th-text-m mt-0.5">Ctrl+V or drag-and-drop starts download instantly</p>
                        </div>
                        <Toggle enabled={settings.quickDownload} onChange={settings.setQuickDownload} />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-th-text mb-2">Double-Click Completed Download</label>
                        <div className="flex gap-2">
                            {([
                                { value: 'open-file' as const, label: 'Open File' },
                                { value: 'open-folder' as const, label: 'Show in Folder' },
                            ]).map(({ value, label }) => (
                                <button
                                    key={value}
                                    onClick={() => settings.setCompletedClickAction(value)}
                                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors border ${
                                        settings.completedClickAction === value
                                            ? 'bg-th-accent/20 border-th-accent text-th-accent-t'
                                            : 'bg-th-raised border-th-border text-th-text-s hover:text-th-text'
                                    }`}
                                >
                                    {label}
                                </button>
                            ))}
                        </div>
                    </div>
                </div>
            </section>
        </div>
    );
};
