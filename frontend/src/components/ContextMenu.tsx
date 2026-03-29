import React, { useEffect, useRef } from 'react';
import { FolderOpen, Play, Pause, Square, Trash2, File, ExternalLink, Copy, RotateCcw } from 'lucide-react';

interface ContextMenuProps {
    x: number;
    y: number;
    visible: boolean;
    onClose: () => void;
    onOpen: () => void;
    onShowInFolder: () => void;
    onCopyLink: () => void;
    onDelete: () => void;
    onRetry: () => void;
    onPause: () => void;
    onResume: () => void;
    onStop: () => void;
    status: string;
}

export const ContextMenu: React.FC<ContextMenuProps> = ({
    x, y, visible, onClose, onOpen, onShowInFolder, onCopyLink, onDelete, onRetry, onPause, onResume, onStop, status
}) => {
    const menuRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                onClose();
            }
        };
        if (visible) {
            document.addEventListener('mousedown', handleClickOutside);
        }
        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
        };
    }, [visible, onClose]);

    if (!visible) return null;

    return (
        <div
            ref={menuRef}
            className="fixed z-50 bg-th-surface border border-th-border rounded-lg shadow-xl py-1 w-48 text-sm text-th-text select-none animate-in fade-in zoom-in-95 duration-75"
            style={{ top: y, left: x }}
        >
            <div
                className="px-4 py-2 hover:bg-th-raised flex items-center gap-2 cursor-pointer"
                onClick={() => { onOpen(); onClose(); }}
            >
                <File size={14} className="text-th-accent-t" /> Open File
            </div>
            <div
                className="px-4 py-2 hover:bg-th-raised flex items-center gap-2 cursor-pointer"
                onClick={() => { onShowInFolder(); onClose(); }}
            >
                <FolderOpen size={14} className="text-yellow-400" /> Show in Folder
            </div>

            <div className="h-px bg-th-border my-1 mx-2"></div>

            {(status === 'downloading' || status === 'probing' || status === 'paused' || status === 'pending' || status === 'scheduled') && (
                <>
                    {(status === 'paused' || status === 'scheduled') ? (
                        <div
                            className="px-4 py-2 hover:bg-th-raised flex items-center gap-2 cursor-pointer"
                            onClick={() => { onResume(); onClose(); }}
                        >
                            <Play size={14} className="text-green-400" /> {status === 'scheduled' ? 'Start Now' : 'Resume'}
                        </div>
                    ) : (
                        <div
                            className="px-4 py-2 hover:bg-th-raised flex items-center gap-2 cursor-pointer"
                            onClick={() => { onPause(); onClose(); }}
                        >
                            <Pause size={14} className="text-yellow-400" /> Pause
                        </div>
                    )}
                    <div
                        className="px-4 py-2 hover:bg-th-raised flex items-center gap-2 cursor-pointer"
                        onClick={() => { onStop(); onClose(); }}
                    >
                        <Square size={14} className="text-th-text-s" /> Stop
                    </div>
                </>
            )}

            <div
                className="px-4 py-2 hover:bg-th-raised flex items-center gap-2 cursor-pointer"
                onClick={() => { onRetry(); onClose(); }}
            >
                <RotateCcw size={14} /> Re-download
            </div>

            <div
                className="px-4 py-2 hover:bg-th-raised flex items-center gap-2 cursor-pointer"
                onClick={() => { onCopyLink(); onClose(); }}
            >
                <Copy size={14} /> Copy Link
            </div>

            <div className="h-px bg-th-border my-1 mx-2"></div>

            <div
                className="px-4 py-2 hover:bg-red-900/30 flex items-center gap-2 cursor-pointer text-red-400"
                onClick={() => { onDelete(); onClose(); }}
            >
                <Trash2 size={14} /> Delete
            </div>
        </div>
    );
};
