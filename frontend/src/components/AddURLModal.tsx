import React, { useState } from 'react';
import { X, Globe, Link2, FolderOpen } from 'lucide-react';

interface AddURLModalProps {
    isOpen: boolean;
    onClose: () => void;
    onAdd: (url: string) => Promise<string>;
}

export const AddURLModal: React.FC<AddURLModalProps> = ({ isOpen, onClose, onAdd }) => {
    const [url, setUrl] = useState("");
    const [error, setError] = useState("");

    if (!isOpen) return null;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        try {
            await onAdd(url);
            setUrl("");
            onClose();
        } catch (err: any) {
            setError(err.message || "Failed to add download");
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
                    <button onClick={onClose} className="p-1 hover:bg-slate-800 rounded-full text-slate-400 hover:text-white transition-colors">
                        <X size={20} />
                    </button>
                </div>

                <form onSubmit={handleSubmit} className="p-6 space-y-6">
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
                            />
                        </div>
                        {error && <p className="text-red-400 text-xs mt-2 flex items-center gap-1">Error: {error}</p>}
                    </div>

                    {/* Options Preview (Collapsed) */}
                    <div className="p-4 bg-slate-800/30 rounded-xl border border-slate-800/50 space-y-3">
                        <div className="flex justify-between items-center text-sm">
                            <span className="text-slate-400 flex items-center gap-2"><FolderOpen size={16} /> Save to</span>
                            <span className="text-slate-300 font-mono text-xs bg-slate-800 px-2 py-1 rounded border border-slate-700">Downloads/</span>
                        </div>
                    </div>

                    <div className="flex justify-end gap-3 pt-2">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-5 py-2.5 rounded-xl font-medium text-slate-400 hover:bg-slate-800 hover:text-white transition-colors"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={!url}
                            className="px-6 py-2.5 bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white rounded-xl font-bold shadow-lg shadow-cyan-900/20 active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            Start Download
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};
