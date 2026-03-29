import React, { useState, useEffect } from 'react';
import { X, Globe, Link2, FolderOpen, AlertTriangle, FileCheck, Copy, DownloadCloud, Loader2, Calendar } from 'lucide-react';
import prettyBytes from 'pretty-bytes';
import { Checkbox } from './common/Checkbox';
import { Dropdown } from './common/Dropdown';
import { useSettingsStore } from '../store';

interface AddURLModalProps {
    isOpen: boolean;
    onClose: () => void;
    onAdd: (url: string, filename?: string, size?: number, path?: string, options?: any) => Promise<string>;
    initialUrl?: string;
}

type ModalStep = 'input' | 'probing' | 'confirm';

export const AddURLModal: React.FC<AddURLModalProps> = ({ isOpen, onClose, onAdd, initialUrl }) => {
    const [url, setUrl] = useState("");
    const [step, setStep] = useState<ModalStep>('input');
    const [error, setError] = useState("");
    const [probeData, setProbeData] = useState<any>(null);
    const [historyConflict, setHistoryConflict] = useState(false);
    const [fileConflict, setFileConflict] = useState<{ exists: boolean, path: string } | null>(null);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [downloadPath, setDownloadPath] = useState("");
    const [savedLocations, setSavedLocations] = useState<any[]>([]);
    const [showPathInput, setShowPathInput] = useState(false);

    // Schedule state
    const [enableSchedule, setEnableSchedule] = useState(false);
    const schedulerTime = useSettingsStore(s => s.schedulerTime);



    // Auto-paste URL from clipboard when modal opens
    useEffect(() => {
        if (isOpen) {
            // Load saved locations
            // @ts-ignore
            if (window.go?.app?.App?.GetDownloadLocations) {
                // @ts-ignore
                window.go.app.App.GetDownloadLocations().then((locs: any) => {
                    setSavedLocations(locs || []);
                });
            }

            if (initialUrl) {
                setUrl(initialUrl);
            } else if (!url && step === 'input') {
                navigator.clipboard.readText().then(text => {
                    const trimmed = text?.trim();
                    if (trimmed && /^https?:\/\/.+/i.test(trimmed)) {
                        setUrl(trimmed);
                    }
                }).catch(() => {
                    // Clipboard access might be denied, ignore
                });
            }
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [isOpen, initialUrl]);

    if (!isOpen) return null;

    const reset = () => {
        setUrl("");
        setStep('input');
        setError("");
        setProbeData(null);
        setHistoryConflict(false);
        setFileConflict(null);
        setIsSubmitting(false);
        setDownloadPath("");
        setShowPathInput(false);
        setEnableSchedule(false);

    };

    const handleClose = () => {
        reset();
        onClose();
    };

    const handleProbe = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!url) return;

        setStep('probing');
        setError("");

        try {
            // 1. Probe
            // @ts-ignore
            const data = await window.go.app.App.ProbeURL(url);
            if (data.status >= 400) {
                throw new Error(`Server returned HTTP ${data.status}`);
            }
            setProbeData(data);

            // 2. History Check
            // @ts-ignore
            const hasHistory = await window.go.app.App.CheckHistory(url);
            setHistoryConflict(hasHistory);

            // 3. Collision Check
            // @ts-ignore
            const collision = await window.go.app.App.CheckCollision(data.filename);
            setFileConflict(collision);

            setStep('confirm');
        } catch (err: any) {
            console.error("Probe error:", err);
            const msg = typeof err === 'string' ? err : (err.message || "Failed to probe URL");
            setError(msg);
            setStep('input');
        }
    };

    const performDownload = async (customFilename?: string) => {
        setIsSubmitting(true);
        try {
            const size = probeData ? probeData.size : 0;
            const finalFilename = customFilename || (probeData ? probeData.filename : undefined);

            const options: any = {};
            if (enableSchedule && schedulerTime) {
                // Compute next occurrence of the global schedule time (HH:MM)
                const [hours, minutes] = schedulerTime.split(':').map(Number);
                const now = new Date();
                const target = new Date();
                target.setHours(hours, minutes, 0, 0);
                if (target <= now) target.setDate(target.getDate() + 1);
                options["start_time"] = target.toISOString();
            }

            const result = await onAdd(url, finalFilename, size, downloadPath, options);

            if (result && result.startsWith("ERROR:")) {
                throw new Error(result.substring(7));
            }
            handleClose();
        } catch (err: any) {
            console.error("Download start error:", err);
            const msg = typeof err === 'string' ? err : (err.message || "Failed to start download");
            setError(msg);
            setIsSubmitting(false);
        }
    };

    const findUniqueFilename = async (original: string) => {
        let name = original;
        let ext = "";
        const parts = original.split('.');
        if (parts.length > 1) {
            ext = "." + parts.pop();
            name = parts.join('.');
        }

        // Try candidate names: filename_2, filename_3, etc.
        for (let i = 2; i < 100; i++) {
            const candidate = `${name}_${i}${ext}`;
            try {
                // @ts-ignore
                const collision = await window.go.app.App.CheckCollision(candidate);
                if (!collision.exists) return candidate;
            } catch (e) {
                return `${name}_${Date.now()}${ext}`;
            }
        }
        return `${name}_${Date.now()}${ext}`;
    };

    const handleSaveAsCopy = async () => {
        if (!probeData?.filename) return;
        setIsSubmitting(true);
        const newName = await findUniqueFilename(probeData.filename);
        performDownload(newName);
    };

    const handlePathChange = (val: string) => {
        if (val === 'custom') {
            setShowPathInput(true);
            setDownloadPath("");
        } else {
            setShowPathInput(false);
            setDownloadPath(val);
        }
    };

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 backdrop-blur-sm animate-fade-in">
            <div className="bg-th-surface w-full max-w-lg rounded-2xl border border-th-border shadow-2xl overflow-hidden transform transition-all scale-100">
                {/* Header */}
                <div className="flex justify-between items-center p-5 border-b border-th-border bg-th-surface/50">
                    <h2 className="text-lg font-bold text-th-text flex items-center gap-2">
                        <Globe className="text-th-accent-t" size={20} />
                        Add New Download
                    </h2>
                    <button onClick={handleClose} className="p-1 hover:bg-th-raised rounded-full text-th-text-s hover:text-th-text transition-colors">
                        <X size={20} />
                    </button>
                </div>

                {step === 'input' || step === 'probing' ? (
                    <form onSubmit={handleProbe} className="p-6 space-y-4">
                        <div>
                            <label className="block text-xs font-semibold text-th-text-s uppercase tracking-wider mb-2">Source URL</label>
                            <div className="relative">
                                <Link2 className="absolute left-3 top-3 text-th-text-m" size={18} />
                                <input
                                    type="text"
                                    autoFocus
                                    placeholder="https://example.com/file.zip"
                                    className="w-full bg-th-base border border-th-border rounded-xl py-3 pl-10 pr-4 text-th-text focus:outline-none focus:border-th-accent focus:ring-1 focus:ring-th-accent transition-all font-mono text-sm shadow-inner"
                                    value={url}
                                    onChange={(e) => setUrl(e.target.value)}
                                    disabled={step === 'probing'}
                                />
                            </div>
                            {error && (
                                <div className="mt-3 p-3 bg-red-950/30 border border-red-900/50 rounded-lg flex items-center gap-2 text-red-400 text-xs">
                                    <AlertTriangle size={14} />
                                    <span>{error}</span>
                                </div>
                            )}

                        </div>

                        <div className="flex justify-end gap-3">
                            <button
                                type="button"
                                onClick={handleClose}
                                className="px-5 py-2.5 rounded-xl font-medium text-th-text-s hover:bg-th-raised hover:text-th-text transition-colors"
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                disabled={!url || step === 'probing'}
                                className="px-6 py-2.5 bg-th-accent hover:bg-th-accent-h text-white rounded-xl font-bold shadow-lg shadow-th-accent/20 active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                            >
                                {step === 'probing' ? <><Loader2 size={16} className="animate-spin" /> Checking...</> : "Next"}
                            </button>
                        </div>
                    </form>
                ) : (
                    <div className="p-6 space-y-6">
                        {/* Summary */}
                        <div className="flex items-start gap-4 p-4 bg-th-raised/50 rounded-xl border border-th-border-s">
                            <div className="p-3 bg-th-overlay/50 rounded-lg">
                                <FileCheck className="text-th-accent-t" size={24} />
                            </div>
                            <div>
                                <h3 className="text-th-text font-medium truncate max-w-[300px]" title={probeData?.filename}>{probeData?.filename}</h3>
                                <div className="flex items-center gap-3 text-xs text-th-text-s mt-1">
                                    <span className="font-mono bg-th-raised px-1.5 py-0.5 rounded">{prettyBytes(probeData?.size || 0)}</span>
                                    <span>•</span>
                                    <span className="truncate max-w-[200px]">{url}</span>
                                </div>
                            </div>
                        </div>

                        {/* Warnings */}
                        <div className="space-y-3">
                            {historyConflict && (
                                <div className="p-3 bg-yellow-900/20 border border-yellow-700/30 rounded-lg flex gap-3">
                                    <AlertTriangle className="text-yellow-500 shrink-0" size={20} />
                                    <div>
                                        <h4 className="text-yellow-500 text-sm font-bold">Already Downloaded</h4>
                                        <p className="text-yellow-200/70 text-xs mt-0.5">You have downloaded this link before.</p>
                                    </div>
                                </div>
                            )}

                            {fileConflict?.exists && (
                                <div className="p-3 bg-orange-900/20 border border-orange-700/30 rounded-lg flex gap-3">
                                    <FolderOpen className="text-orange-500 shrink-0" size={20} />
                                    <div>
                                        <h4 className="text-orange-500 text-sm font-bold">File Exists on Disk</h4>
                                        <p className="text-orange-200/70 text-xs mt-0.5">
                                            A file named <span className="font-mono text-orange-100">{probeData?.filename}</span> already exists in the destination.
                                        </p>
                                    </div>
                                </div>
                            )}
                        </div>

                        {/* Schedule Option */}
                        <div className="bg-th-raised/50 p-3 rounded-xl border border-th-border-s">
                            <div className="flex items-center gap-2">
                                <Checkbox
                                    id="schedule"
                                    checked={enableSchedule}
                                    onChange={setEnableSchedule}
                                />
                                <label htmlFor="schedule" className="text-sm font-medium text-th-text-s select-none cursor-pointer flex items-center gap-2">
                                    <Calendar size={14} className="text-th-text-s" />
                                    Schedule download
                                </label>
                                {enableSchedule && (
                                    <span className="ml-auto text-xs text-purple-400">Scheduled for {schedulerTime}</span>
                                )}
                            </div>
                        </div>

                        {/* Path Selector */}
                        <div className="bg-th-raised/50 p-3 rounded-xl border border-th-border-s space-y-2">
                            <div className="flex justify-between items-center text-xs text-th-text-s uppercase font-semibold tracking-wider">
                                <span>Save Location</span>
                            </div>

                            {!showPathInput ? (
                                <Dropdown
                                    value={downloadPath}
                                    onChange={handlePathChange}
                                    options={[
                                        { value: '', label: 'Default Downloads' },
                                        ...savedLocations.map((loc: any) => ({ value: loc.path, label: loc.nickname || loc.path })),
                                        { value: 'custom', label: '+ Custom Path...' },
                                    ]}
                                />
                            ) : (
                                <div className="flex gap-2">
                                    <input
                                        type="text"
                                        autoFocus
                                        placeholder="C:\Downloads\MyFolder"
                                        className="flex-1 bg-th-surface border border-th-border-s rounded-lg p-2 text-sm text-th-text focus:outline-none focus:border-th-accent font-mono"
                                        value={downloadPath}
                                        onChange={e => setDownloadPath(e.target.value)}
                                    />
                                    <button
                                        onClick={() => setShowPathInput(false)}
                                        className="px-3 bg-th-raised hover:bg-th-overlay border border-th-border-s rounded-lg text-th-text-s"
                                    >
                                        Cancel
                                    </button>
                                </div>
                            )}
                        </div>

                        {/* Actions */}
                        <div className="flex flex-col gap-3 pt-2">
                            {!fileConflict?.exists ? (
                                <button
                                    onClick={() => performDownload()}
                                    disabled={isSubmitting}
                                    className="w-full py-3 bg-green-600 hover:bg-green-500 text-white rounded-xl font-bold transition-all shadow-lg shadow-green-900/20 flex justify-center items-center gap-2"
                                >
                                    {isSubmitting ? <Loader2 className="animate-spin" /> : <><DownloadCloud size={18} /> {historyConflict ? "Download Again" : (enableSchedule ? "Schedule Download" : "Start Download")}</>}
                                </button>
                            ) : (
                                <div className="grid grid-cols-2 gap-3">
                                    <button
                                        onClick={() => handleClose()}
                                        className="py-3 bg-th-raised hover:bg-th-overlay text-th-text rounded-xl font-medium transition-colors border border-th-border-s"
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        onClick={handleSaveAsCopy}
                                        disabled={isSubmitting}
                                        className="py-3 bg-th-accent hover:bg-th-accent-h text-white rounded-xl font-bold transition-colors shadow-lg shadow-th-accent/20 flex justify-center items-center gap-2"
                                    >
                                        {isSubmitting ? <Loader2 className="animate-spin" /> : <><Copy size={18} /> Save as Copy</>}
                                    </button>
                                </div>
                            )}

                            {/* Back to Input */}
                            <button onClick={() => setStep('input')} className="text-xs text-th-text-m hover:text-th-text-s py-2">
                                Change URL
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};
