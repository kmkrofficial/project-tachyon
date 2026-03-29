import React, { useState } from 'react';
import { X, AlertTriangle, Trash2 } from 'lucide-react';

type ClearAction = 'completed' | 'failed' | 'all' | 'all-with-files';

interface ClearDownloadsModalProps {
    isOpen: boolean;
    onClose: () => void;
    onClear: (action: ClearAction) => void;
    context: 'dashboard' | 'scheduler';
}

export const ClearDownloadsModal: React.FC<ClearDownloadsModalProps> = ({ isOpen, onClose, onClear, context }) => {
    const [confirmDelete, setConfirmDelete] = useState(false);

    if (!isOpen) return null;

    const handleClose = () => {
        setConfirmDelete(false);
        onClose();
    };

    const handleAction = (action: ClearAction) => {
        if (action === 'all-with-files') {
            if (!confirmDelete) {
                setConfirmDelete(true);
                return;
            }
        }
        onClear(action);
        setConfirmDelete(false);
        onClose();
    };

    return (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 backdrop-blur-sm animate-fade-in">
            <div className="bg-th-surface w-full max-w-md rounded-2xl border border-th-border shadow-2xl overflow-hidden">
                <div className="flex justify-between items-center p-5 border-b border-th-border bg-th-surface/50">
                    <h2 className="text-lg font-bold text-th-text flex items-center gap-2">
                        <Trash2 className="text-red-400" size={20} />
                        Clear Downloads
                    </h2>
                    <button onClick={handleClose} className="p-1 hover:bg-th-raised rounded-full text-th-text-s hover:text-th-text transition-colors">
                        <X size={20} />
                    </button>
                </div>

                <div className="p-5 space-y-2">
                    <button
                        onClick={() => handleAction('completed')}
                        className="w-full text-left px-4 py-3 rounded-lg hover:bg-th-raised transition-colors text-sm text-th-text flex items-center justify-between group"
                    >
                        <span>Clear Completed</span>
                        <span className="text-xs text-th-text-m group-hover:text-green-400">Remove finished downloads from list</span>
                    </button>

                    {context === 'dashboard' && (
                        <button
                            onClick={() => handleAction('failed')}
                            className="w-full text-left px-4 py-3 rounded-lg hover:bg-th-raised transition-colors text-sm text-th-text flex items-center justify-between group"
                        >
                            <span>Clear Failed</span>
                            <span className="text-xs text-th-text-m group-hover:text-orange-400">Remove errored downloads from list</span>
                        </button>
                    )}

                    <div className="h-px bg-th-border my-2" />

                    <button
                        onClick={() => handleAction('all')}
                        className="w-full text-left px-4 py-3 rounded-lg hover:bg-th-raised transition-colors text-sm text-th-text flex items-center justify-between group"
                    >
                        <span>Clear All</span>
                        <span className="text-xs text-th-text-m group-hover:text-red-400">Remove all from list (keep files)</span>
                    </button>

                    {!confirmDelete ? (
                        <button
                            onClick={() => handleAction('all-with-files')}
                            className="w-full text-left px-4 py-3 rounded-lg hover:bg-red-500/10 border border-transparent hover:border-red-500/20 transition-colors text-sm text-red-400 flex items-center justify-between group"
                        >
                            <span className="flex items-center gap-2">
                                <AlertTriangle size={14} />
                                Clear All + Delete Files
                            </span>
                            <span className="text-xs text-th-text-m group-hover:text-red-400">Permanently delete from disk</span>
                        </button>
                    ) : (
                        <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-4 space-y-3">
                            <p className="text-sm text-red-400 flex items-center gap-2 font-medium">
                                <AlertTriangle size={16} />
                                This will permanently delete all files from disk. This cannot be undone.
                            </p>
                            <div className="flex items-center gap-2">
                                <button
                                    onClick={() => setConfirmDelete(false)}
                                    className="flex-1 px-3 py-2 rounded-lg bg-th-raised text-th-text text-sm hover:bg-th-overlay transition-colors"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={() => handleAction('all-with-files')}
                                    className="flex-1 px-3 py-2 rounded-lg bg-red-500 text-white text-sm font-medium hover:bg-red-600 transition-colors"
                                >
                                    Yes, Delete Everything
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};
