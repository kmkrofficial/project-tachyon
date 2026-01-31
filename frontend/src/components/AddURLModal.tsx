import React, { useState, useEffect } from 'react';
import { X, Globe, Link2, FolderOpen, AlertTriangle, FileCheck, Copy, DownloadCloud, Loader2, Calendar, Clock, Settings2 } from 'lucide-react';
import prettyBytes from 'pretty-bytes';

interface AddURLModalProps {
    isOpen: boolean;
    onClose: () => void;
    onAdd: (url: string, filename?: string, size?: number, path?: string, options?: any) => Promise<string>;
}

type ModalStep = 'input' | 'probing' | 'confirm';

export const AddURLModal: React.FC<AddURLModalProps> = ({ isOpen, onClose, onAdd }) => {
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
    const [scheduleTime, setScheduleTime] = useState("");
    const [enableSchedule, setEnableSchedule] = useState(false);

    // Advanced Options
    const [showAdvanced, setShowAdvanced] = useState(false);
    const [headers, setHeaders] = useState("");
    const [cookies, setCookies] = useState("");
    const [userAgent, setUserAgent] = useState("");

    // Auto-paste URL from clipboard when modal opens
    useEffect(() => {
        if (isOpen) {
            // Load saved locations
            // @ts-ignore
            if (window.go?.main?.App?.GetDownloadLocations) {
                // @ts-ignore
                window.go.main.App.GetDownloadLocations().then((locs: any) => {
                    setSavedLocations(locs || []);
                });
            }

            if (!url && step === 'input') {
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
    }, [isOpen]);

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
        setScheduleTime("");
        setShowAdvanced(false);
        setHeaders("");
        setCookies("");
        setUserAgent("");
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
            const data = await window.go.main.App.ProbeURL(url);
            if (data.status >= 400) {
                throw new Error(`Server returned HTTP ${data.status}`);
            }
            setProbeData(data);

            // 2. History Check
            // @ts-ignore
            const hasHistory = await window.go.main.App.CheckHistory(url);
            setHistoryConflict(hasHistory);

            // 3. Collision Check
            // @ts-ignore
            const collision = await window.go.main.App.CheckCollision(data.filename);
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
            if (enableSchedule && scheduleTime) {
                const date = new Date(scheduleTime);
                options["start_time"] = date.toISOString();
            }

            if (headers) options["headers"] = headers;
            if (cookies) options["cookies"] = cookies;
            if (userAgent) options["user_agent"] = userAgent;

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
                const collision = await window.go.main.App.CheckCollision(candidate);
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

    const handlePathChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
        const val = e.target.value;
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
            <div className="bg-slate-900 w-full max-w-lg rounded-2xl border border-slate-800 shadow-2xl overflow-hidden transform transition-all scale-100">
                {/* Header */}
                <div className="flex justify-between items-center p-5 border-b border-slate-800 bg-slate-900/50">
                    <h2 className="text-lg font-bold text-white flex items-center gap-2">
                        <Globe className="text-cyan-500" size={20} />
                        Add New Download
                    </h2>
                    <button onClick={handleClose} className="p-1 hover:bg-slate-800 rounded-full text-slate-400 hover:text-white transition-colors">
                        <X size={20} />
                    </button>
                </div>

                {step === 'input' || step === 'probing' ? (
                    <form onSubmit={handleProbe} className="p-6 space-y-6">
                        <div>
                            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2">Source URL</label>
                            <div className="relative">
                                <Link2 className="absolute left-3 top-3 text-slate-500" size={18} />
                                <input
                                    type="text"
                                    autoFocus
                                    placeholder="https://example.com/file.zip"
                                    className="w-full bg-slate-950 border border-slate-800 rounded-xl py-3 pl-10 pr-4 text-slate-200 focus:outline-none focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500 transition-all font-mono text-sm shadow-inner"
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

                        {/* Advanced Options Toggle */}
                        <div>
                            <button
                                type="button"
                                onClick={() => setShowAdvanced(!showAdvanced)}
                                className="flex items-center gap-2 text-xs font-semibold text-slate-400 hover:text-slate-300 transition-colors uppercase tracking-wider"
                            >
                                <Settings2 size={14} />
                                {showAdvanced ? "Hide Advanced Options" : "Show Advanced Options"}
                            </button>

                            {showAdvanced && (
                                <div className="mt-4 space-y-4 p-4 bg-slate-950/50 rounded-xl border border-slate-800 animate-slide-down">
                                    <div>
                                        <label className="block text-xs text-slate-500 mb-1.5">User Agent</label>
                                        <input
                                            type="text"
                                            className="w-full bg-slate-900 border border-slate-700 rounded-lg p-2 text-xs text-slate-300 focus:outline-none focus:border-cyan-500 font-mono placeholder:text-slate-700"
                                            placeholder="Mozilla/5.0..."
                                            value={userAgent}
                                            onChange={e => setUserAgent(e.target.value)}
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-xs text-slate-500 mb-1.5">Referer / Custom Headers (JSON)</label>
                                        <textarea
                                            className="w-full bg-slate-900 border border-slate-700 rounded-lg p-2 text-xs text-slate-300 focus:outline-none focus:border-cyan-500 font-mono h-20 placeholder:text-slate-700"
                                            placeholder='{"Referer": "https://source.com", "Authorization": "Bearer..."}'
                                            value={headers}
                                            onChange={e => setHeaders(e.target.value)}
                                        />
                                    </div>
                                    <div>
                                        <label className="block text-xs text-slate-500 mb-1.5">Cookies</label>
                                        <textarea
                                            className="w-full bg-slate-900 border border-slate-700 rounded-lg p-2 text-xs text-slate-300 focus:outline-none focus:border-cyan-500 font-mono h-20 placeholder:text-slate-700"
                                            placeholder="key=value; key2=value2"
                                            value={cookies}
                                            onChange={e => setCookies(e.target.value)}
                                        />
                                    </div>
                                </div>
                            )}
                        </div>

                        <div className="flex justify-end gap-3 pt-2">
                            <button
                                type="button"
                                onClick={handleClose}
                                className="px-5 py-2.5 rounded-xl font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition-colors"
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                disabled={!url || step === 'probing'}
                                className="px-6 py-2.5 bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white rounded-xl font-bold shadow-lg shadow-cyan-900/20 active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                            >
                                {step === 'probing' ? <><Loader2 size={16} className="animate-spin" /> Checking...</> : "Next"}
                            </button>
                        </div>
                    </form>
                ) : (
                    <div className="p-6 space-y-6">
                        {/* Summary */}
                        <div className="flex items-start gap-4 p-4 bg-slate-800/50 rounded-xl border border-slate-700">
                            <div className="p-3 bg-slate-700/50 rounded-lg">
                                <FileCheck className="text-cyan-400" size={24} />
                            </div>
                            <div>
                                <h3 className="text-white font-medium truncate max-w-[300px]" title={probeData?.filename}>{probeData?.filename}</h3>
                                <div className="flex items-center gap-3 text-xs text-slate-400 mt-1">
                                    <span className="font-mono bg-slate-800 px-1.5 py-0.5 rounded">{prettyBytes(probeData?.size || 0)}</span>
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
                        <div className="bg-slate-800/50 p-3 rounded-xl border border-slate-700 space-y-2">
                            <div className="flex items-center gap-2">
                                <input
                                    type="checkbox"
                                    id="schedule"
                                    className="w-4 h-4 rounded border-slate-600 bg-slate-700 text-cyan-500 focus:ring-cyan-500"
                                    checked={enableSchedule}
                                    onChange={e => setEnableSchedule(e.target.checked)}
                                />
                                <label htmlFor="schedule" className="text-sm font-medium text-slate-300 select-none cursor-pointer flex items-center gap-2">
                                    <Calendar size={14} className="text-slate-400" />
                                    Start Later
                                </label>
                            </div>

                            {enableSchedule && (
                                <div className="pl-6">
                                    <input
                                        type="datetime-local"
                                        className="w-full bg-slate-900 border border-slate-700 rounded-lg p-2 text-sm text-slate-200 focus:outline-none focus:border-cyan-500"
                                        value={scheduleTime}
                                        onChange={e => setScheduleTime(e.target.value)}
                                    />
                                    <p className="text-[10px] text-slate-500 mt-1 flex items-center gap-1">
                                        <Clock size={10} />
                                        Task will be queued and auto-started at this time.
                                    </p>
                                </div>
                            )}
                        </div>

                        {/* Path Selector */}
                        <div className="bg-slate-800/50 p-3 rounded-xl border border-slate-700 space-y-2">
                            <div className="flex justify-between items-center text-xs text-slate-400 uppercase font-semibold tracking-wider">
                                <span>Save Location</span>
                            </div>

                            {!showPathInput ? (
                                <select
                                    className="w-full bg-slate-900 border border-slate-700 rounded-lg p-2 text-sm text-slate-200 focus:outline-none focus:border-cyan-500"
                                    value={downloadPath}
                                    onChange={handlePathChange}
                                >
                                    <option value="">Default Downloads</option>
                                    {savedLocations.map((loc: any) => (
                                        <option key={loc.path} value={loc.path}>{loc.nickname || loc.path}</option>
                                    ))}
                                    <option value="custom">+ Custom Path...</option>
                                </select>
                            ) : (
                                <div className="flex gap-2">
                                    <input
                                        type="text"
                                        autoFocus
                                        placeholder="C:\Downloads\MyFolder"
                                        className="flex-1 bg-slate-900 border border-slate-700 rounded-lg p-2 text-sm text-slate-200 focus:outline-none focus:border-cyan-500 font-mono"
                                        value={downloadPath}
                                        onChange={e => setDownloadPath(e.target.value)}
                                    />
                                    <button
                                        onClick={() => setShowPathInput(false)}
                                        className="px-3 bg-slate-800 hover:bg-slate-700 border border-slate-700 rounded-lg text-slate-400"
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
                                        className="py-3 bg-slate-800 hover:bg-slate-700 text-white rounded-xl font-medium transition-colors border border-slate-700"
                                    >
                                        Cancel
                                    </button>
                                    <button
                                        onClick={handleSaveAsCopy}
                                        disabled={isSubmitting}
                                        className="py-3 bg-cyan-600 hover:bg-cyan-500 text-white rounded-xl font-bold transition-colors shadow-lg shadow-cyan-900/20 flex justify-center items-center gap-2"
                                    >
                                        {isSubmitting ? <Loader2 className="animate-spin" /> : <><Copy size={18} /> Save as Copy</>}
                                    </button>
                                </div>
                            )}

                            {/* Back to Input */}
                            <button onClick={() => setStep('input')} className="text-xs text-slate-500 hover:text-slate-300 py-2">
                                Change URL
                            </button>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};
