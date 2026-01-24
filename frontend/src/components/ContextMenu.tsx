import React, { useEffect, useRef } from 'react';
import { FolderOpen, Play, Pause, Trash2, File, ExternalLink, Copy } from 'lucide-react';

interface ContextMenuProps {
    x: number;
    y: number;
    visible: boolean;
    onClose: () => void;
    onOpen: () => void;
    onShowInFolder: () => void;
    onCopyLink: () => void;
    onDelete: () => void;
    status: string;
}

export const ContextMenu: React.FC<ContextMenuProps> = ({
    x, y, visible, onClose, onOpen, onShowInFolder, onCopyLink, onDelete, status
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
            className="fixed z-50 bg-gray-800 border border-gray-700 rounded-lg shadow-xl py-1 w-48 text-sm text-gray-200 select-none"
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
