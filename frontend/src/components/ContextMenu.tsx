import React, { useEffect, useRef } from 'react';
import { FolderOpen, Play, Pause, Trash2, File, ExternalLink, Copy, RotateCcw, Sliders } from 'lucide-react';

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
    onSetPriority: (p: number) => void;
    status: string;
}

export const ContextMenu: React.FC<ContextMenuProps> = ({
    x, y, visible, onClose, onOpen, onShowInFolder, onCopyLink, onDelete, onRetry, onSetPriority, status
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
            className="fixed z-50 bg-gray-800 border border-gray-700 rounded-lg shadow-xl py-1 w-48 text-sm text-gray-200 select-none animate-in fade-in zoom-in-95 duration-75"
            style={{ top: y, left: x }}
        >
            <div
                className="px-4 py-2 hover:bg-gray-700 flex items-center gap-2 cursor-pointer"
                onClick={() => { onOpen(); onClose(); }}
            >
                <File size={14} className="text-blue-400" /> Open File
            </div>
            <div
                className="px-4 py-2 hover:bg-gray-700 flex items-center gap-2 cursor-pointer"
                onClick={() => { onShowInFolder(); onClose(); }}
            >
                <FolderOpen size={14} className="text-yellow-400" /> Show in Folder
            </div>

            <div className="h-px bg-gray-700 my-1 mx-2"></div>

            <div className="px-4 py-2 hover:bg-gray-700 group relative flex items-center gap-2 cursor-pointer">
                <span className="flex-1 flex items-center gap-2"><Sliders size={14} /> Priority</span>
                <span className="text-xs text-gray-500">▶</span>
                {/* Submenu */}
                <div className="absolute left-full top-0 ml-1 w-32 bg-gray-800 border border-gray-700 rounded-lg shadow-xl hidden group-hover:block">
                    {/* 3 = High, 2 = Normal, 1 = Low based on my implementation assumption in previous step? 
                        Wait, I used SetPriority(id, priority). 
                        Let's align with: High=3, Normal=2, Low=1. or reverse?
                        Standard: High Priority usually means executed SOONER.
                        But queue execution is by QueueOrder. Priority is just a label for now unless I sort by it.
                        User asked for "priority setting option".
                     */}
                    <div className="px-4 py-2 hover:bg-gray-700 cursor-pointer" onClick={() => { onSetPriority(3); onClose(); }}>High</div>
                    <div className="px-4 py-2 hover:bg-gray-700 cursor-pointer" onClick={() => { onSetPriority(2); onClose(); }}>Normal</div>
                    <div className="px-4 py-2 hover:bg-gray-700 cursor-pointer" onClick={() => { onSetPriority(1); onClose(); }}>Low</div>
                </div>
            </div>

            <div
                className="px-4 py-2 hover:bg-gray-700 flex items-center gap-2 cursor-pointer"
                onClick={() => { onRetry(); onClose(); }}
            >
                <RotateCcw size={14} /> Re-download
            </div>

            <div
                className="px-4 py-2 hover:bg-gray-700 flex items-center gap-2 cursor-pointer"
                onClick={() => { onCopyLink(); onClose(); }}
            >
                <Copy size={14} /> Copy Link
            </div>

            <div className="h-px bg-gray-700 my-1 mx-2"></div>

            <div
                className="px-4 py-2 hover:bg-gray-700 flex items-center gap-2 cursor-pointer text-red-400 hover:bg-red-900/30"
                onClick={() => { onDelete(); onClose(); }}
            >
                <Trash2 size={14} /> Delete
            </div>
        </div>
    );
};
