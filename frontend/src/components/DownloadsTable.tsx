import React, { useState, useCallback } from 'react';
import { DownloadItem } from '../types';
import { File, FileVideo, FileArchive, CheckCircle, AlertCircle, Pause, Play, Trash, Folder, RotateCcw, AlertTriangle, CheckSquare } from 'lucide-react';
import { Checkbox } from './common/Checkbox';
import prettyBytes from 'pretty-bytes';
import { cn } from '../utils';
import { ContextMenu } from './ContextMenu';
import { DownloadDetailPanel } from './DownloadDetailPanel';
import { useSettingsStore } from '../store';

interface DownloadsTableProps {
    data: DownloadItem[];
    onOpenFile: (id: string) => void;
    onOpenFolder: (id: string) => void;
    onReorder: (id: string, direction: string) => void;
    onSetPriority: (id: string, priority: number) => void;
    addToast: (type: 'success' | 'error' | 'warning' | 'info', title: string, message: string) => void;
    selectedIds: Set<string>;
    onSelectionChange: (ids: Set<string>) => void;
}

const getFileIcon = (filename: string) => {
    const ext = filename.split('.').pop()?.toLowerCase() || '';
    if (['mp4', 'mkv', 'webm', 'avi', 'mov', 'flv', 'wmv'].includes(ext)) return FileVideo;
    if (['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz'].includes(ext)) return FileArchive;
    return File;
};

const statusColors: Record<string, string> = {
    downloading: "bg-th-accent/10 text-th-accent-t border-th-accent/20",
    completed: "bg-green-500/10 text-green-500 border-green-500/20",
    error: "bg-red-500/10 text-red-500 border-red-500/20",
    paused: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20",
    pending: "bg-th-raised text-th-text-s border-th-border",
    probing: "bg-th-accent/10 text-th-accent-t border-th-accent/20",
    stopped: "bg-th-raised text-th-text-s border-th-border",
};

const formatEta = (item: DownloadItem): string => {
    const { status, size, progress, speed_MBs } = item;
    if (status !== 'downloading') return status === 'completed' ? 'Done' : '';
    if (!speed_MBs || speed_MBs <= 0) return '∞';
    const remaining = size * (1 - progress / 100);
    const seconds = remaining / (speed_MBs * 1024 * 1024);
    if (seconds < 60) return `${Math.ceil(seconds)}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${Math.ceil(seconds % 60)}s`;
    return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
};

export const DownloadsTable: React.FC<DownloadsTableProps> = ({ data, onOpenFile, onOpenFolder, onReorder, onSetPriority, addToast, selectedIds, onSelectionChange }) => {
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; id: string | null }>({ x: 0, y: 0, id: null });
    const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
    const [deleteFile, setDeleteFile] = useState(false);
    const [bulkDeleteConfirm, setBulkDeleteConfirm] = useState(false);
    const [bulkDeleteFile, setBulkDeleteFile] = useState(false);
    const [selectMode, setSelectMode] = useState(false);
    const [detailId, setDetailId] = useState<string | null>(null);

    const handlePause = useCallback(async (id: string) => {
        try { await window.go.app.App.PauseDownload(id); } catch (e) { console.error(e); }
    }, []);

    const handleResume = useCallback(async (id: string) => {
        try { await window.go.app.App.ResumeDownload(id); } catch (e: any) {
            console.error(e);
            addToast('error', 'Resume Failed', typeof e === 'string' ? e : e.message);
        }
    }, [addToast]);

    const handleStop = useCallback(async (id: string) => {
        try { await window.go.app.App.StopDownload(id); } catch (e) { console.error(e); }
    }, []);

    const handleRetry = useCallback(async (url: string) => {
        try {
            await window.go.app.App.AddDownload(url);
            addToast('info', 'Re-downloading', 'Added download back to queue');
        } catch (e: any) {
            console.error(e);
            addToast('error', 'Retry Failed', typeof e === 'string' ? e : e.message);
        }
    }, [addToast]);

    const handleDelete = async (id: string, withFile: boolean) => {
        try {
            await window.go.app.App.DeleteDownload(id, withFile);
            setDeleteConfirmId(null);
        } catch (e: any) {
            console.error(e);
            if (typeof e === 'string' && e.startsWith("WARNING:")) {
                setDeleteConfirmId(null);
                addToast('warning', 'File Deletion Failed', e.replace("WARNING: ", ""));
            } else {
                addToast('error', 'Delete Failed', typeof e === 'string' ? e : e.message);
            }
        }
    };

    const toggleSelect = useCallback((id: string) => {
        const next = new Set(selectedIds);
        if (next.has(id)) next.delete(id); else next.add(id);
        onSelectionChange(next);
    }, [selectedIds, onSelectionChange]);

    const toggleAll = useCallback(() => {
        if (selectedIds.size === data.length && data.length > 0) onSelectionChange(new Set());
        else onSelectionChange(new Set(data.map(d => d.id)));
    }, [selectedIds, data, onSelectionChange]);

    const exitSelectMode = useCallback(() => {
        setSelectMode(false);
        onSelectionChange(new Set());
    }, [onSelectionChange]);

    const handleBulkDelete = async () => {
        for (const id of selectedIds) {
            try { await window.go.app.App.DeleteDownload(id, bulkDeleteFile); } catch (e: any) {
                if (typeof e === 'string' && e.startsWith("WARNING:"))
                    addToast('warning', 'File Deletion Failed', e.replace("WARNING: ", ""));
                else addToast('error', 'Delete Failed', typeof e === 'string' ? e : e.message);
            }
        }
        onSelectionChange(new Set());
        setBulkDeleteConfirm(false);
        setBulkDeleteFile(false);
    };

    const handleBulkPause = useCallback(async () => {
        for (const id of selectedIds) { try { await window.go.app.App.PauseDownload(id); } catch {} }
    }, [selectedIds]);

    const handleBulkResume = useCallback(async () => {
        for (const id of selectedIds) { try { await window.go.app.App.ResumeDownload(id); } catch {} }
    }, [selectedIds]);

    const handleContextMenu = (e: React.MouseEvent, id: string) => {
        e.preventDefault();
        setContextMenu({ x: e.clientX, y: e.clientY, id });
    };

    const handleCopyLink = (url: string) => {
        navigator.clipboard.writeText(url);
        addToast('success', 'Copied', 'Link copied to clipboard');
    };

    const contextItem = data.find(d => d.id === contextMenu.id);

    return (
        <>
            {/* Select mode bar - only visible when selecting */}
            {selectMode && (
                <div className="flex items-center px-3 py-1.5 border-b border-th-border bg-th-surface/95 shrink-0">
                    <div className="flex items-center gap-2 w-full">
                        <Checkbox checked={data.length > 0 && selectedIds.size === data.length}
                            indeterminate={selectedIds.size > 0 && selectedIds.size < data.length}
                            onChange={toggleAll} size="sm" />
                        <span className="text-[11px] text-th-text-m">{selectedIds.size} of {data.length} selected</span>
                        <button onClick={exitSelectMode} className="ml-auto text-[11px] text-th-text-s hover:text-th-text transition-colors">Cancel</button>
                    </div>
                </div>
            )}

            {/* Tabular download list */}
            <div className="flex-1 min-h-0 overflow-auto scrollbar-thin scrollbar-thumb-th-raised scrollbar-track-transparent">
                <table className="w-full text-left border-collapse">
                    <thead className="bg-th-surface/95 sticky top-0 z-30 backdrop-blur-sm border-b border-th-border">
                        <tr>
                            {selectMode && <th className="w-8 px-3 py-2" />}
                            <th className="px-3 py-2 text-[11px] font-bold text-th-text-m uppercase tracking-wider">
                                <div className="flex items-center gap-1.5">
                                    {!selectMode && data.length > 0 && (
                                        <button onClick={() => setSelectMode(true)} className="text-th-text-m hover:text-th-text transition-colors" title="Select">
                                            <CheckSquare size={13} />
                                        </button>
                                    )}
                                    File Name
                                </div>
                            </th>
                            <th className="px-3 py-2 text-[11px] font-bold text-th-text-m uppercase tracking-wider w-24">Status</th>
                            <th className="px-3 py-2 text-[11px] font-bold text-th-text-m uppercase tracking-wider w-36 hidden sm:table-cell">Progress</th>
                            <th className="px-3 py-2 text-[11px] font-bold text-th-text-m uppercase tracking-wider w-20 hidden md:table-cell">Size</th>
                            <th className="px-3 py-2 text-[11px] font-bold text-th-text-m uppercase tracking-wider w-24 hidden md:table-cell">Speed</th>
                            <th className="px-3 py-2 text-[11px] font-bold text-th-text-m uppercase tracking-wider w-16 hidden lg:table-cell">ETA</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-th-border">
                        {data.map(item => {
                            const Icon = getFileIcon(item.filename);
                            const eta = formatEta(item);
                            const isMissing = item.status === 'completed' && item.file_exists === false;
                            const isSelected = selectedIds.has(item.id);

                            return (
                                <tr
                                    key={item.id}
                                    className={cn(
                                        "group hover:bg-th-raised/50 transition-colors cursor-default",
                                        isSelected ? "bg-th-accent/10" : "",
                                        detailId === item.id && !selectMode ? "bg-th-accent/5 ring-1 ring-inset ring-th-accent/20" : ""
                                    )}
                                    onContextMenu={e => handleContextMenu(e, item.id)}
                                    onClick={() => {
                                        if (selectMode) { toggleSelect(item.id); return; }
                                        setDetailId(prev => prev === item.id ? null : item.id);
                                    }}
                                    onDoubleClick={() => {
                                        if (item.status === 'completed' && !selectMode) {
                                            const action = useSettingsStore.getState().completedClickAction;
                                            action === 'open-folder' ? onOpenFolder(item.id) : onOpenFile(item.id);
                                        }
                                    }}
                                >
                                    {selectMode && (
                                        <td className="px-3 py-1.5">
                                            <Checkbox checked={isSelected}
                                                onChange={() => toggleSelect(item.id)}
                                                onClick={e => e.stopPropagation()} size="sm" />
                                        </td>
                                    )}
                                    <td className="px-3 py-1.5">
                                        <div className="flex items-center gap-2 min-w-0">
                                            <Icon size={14} className="text-th-text-s shrink-0" />
                                            <span className="text-[13px] font-medium text-th-text truncate">{item.filename}</span>
                                        </div>
                                    </td>
                                    <td className="px-3 py-1.5">
                                        <div className="flex items-center gap-1.5">
                                            <span className={cn(
                                                "px-1.5 py-px rounded text-[10px] font-semibold border capitalize flex items-center gap-1",
                                                statusColors[item.status] || statusColors.pending,
                                                isMissing ? "opacity-50" : ""
                                            )}>
                                                {(item.status === 'downloading' || item.status === 'probing') && <span className="w-1 h-1 rounded-full bg-th-accent-t animate-pulse" />}
                                                {item.status === 'error' && <AlertCircle size={10} />}
                                                {item.status}
                                            </span>
                                            {isMissing && (
                                                <span className="px-1 py-px rounded text-[9px] font-bold border border-red-500/30 bg-red-500/10 text-red-400 uppercase flex items-center gap-0.5">
                                                    <AlertTriangle size={9} /> Missing
                                                </span>
                                            )}
                                        </div>
                                    </td>
                                    <td className="px-3 py-1.5 hidden sm:table-cell">
                                        <div className="flex items-center gap-2">
                                            <div className="h-1 bg-th-raised rounded-full overflow-hidden flex-1 relative">
                                                <div
                                                    className={cn(
                                                        "h-full rounded-full transition-all duration-500 ease-out",
                                                        item.status === 'completed' ? "bg-green-500" :
                                                            item.status === 'error' ? "bg-red-500" : "bg-th-accent"
                                                    )}
                                                    style={{ width: `${item.progress}%` }}
                                                >
                                                    {item.status === 'downloading' && (
                                                        <div className="absolute inset-0 bg-white/20 animate-[shimmer_1s_infinite]" style={{ backgroundImage: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.4), transparent)' }} />
                                                    )}
                                                </div>
                                            </div>
                                            <span className="text-[11px] text-th-text-s tabular-nums w-8 text-right shrink-0">{item.progress.toFixed(0)}%</span>
                                        </div>
                                    </td>
                                    <td className="px-3 py-1.5 hidden md:table-cell">
                                        <span className="text-[11px] text-th-text-m">{prettyBytes(item.size || 0)}</span>
                                    </td>
                                    <td className="px-3 py-1.5 hidden md:table-cell">
                                        {item.status === 'downloading' ? (
                                            <span className="text-[11px] font-mono text-th-accent-t">{(item.speed_MBs || 0).toFixed(1)} MB/s</span>
                                        ) : (
                                            <span className="text-[11px] text-th-text-m">-</span>
                                        )}
                                    </td>
                                    <td className="px-3 py-1.5 hidden lg:table-cell">
                                        <span className="text-[11px] text-th-text-m font-mono">{eta || '-'}</span>
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>

                {data.length === 0 && (
                    <div className="flex flex-col items-center justify-center h-full min-h-[300px] text-th-text-m">
                        <div className="w-14 h-14 bg-th-raised/50 rounded-full flex items-center justify-center mb-3">
                            <File size={28} className="opacity-50" />
                        </div>
                        <p className="font-medium text-lg">No downloads yet</p>
                        <p className="text-sm text-center max-w-xs">Drag + drop a link, press Ctrl+V, or use the Add Download button to get started</p>
                    </div>
                )}
            </div>

            {/* Bulk Action Bar */}
            {selectMode && selectedIds.size > 0 && (
                <div className="shrink-0 border-t border-th-border bg-th-surface px-3 py-1.5 flex items-center gap-2">
                    <span className="text-xs text-th-text font-medium">{selectedIds.size} selected</span>
                    <div className="flex items-center gap-1 ml-auto">
                        <button onClick={handleBulkPause} className="px-2.5 py-1 text-xs rounded bg-th-raised hover:bg-th-overlay text-th-text-s hover:text-th-text transition-colors flex items-center gap-1">
                            <Pause size={12} /> Pause
                        </button>
                        <button onClick={handleBulkResume} className="px-2.5 py-1 text-xs rounded bg-th-raised hover:bg-th-overlay text-th-text-s hover:text-th-text transition-colors flex items-center gap-1">
                            <Play size={12} /> Resume
                        </button>
                        <button onClick={() => setBulkDeleteConfirm(true)} className="px-2.5 py-1 text-xs rounded bg-red-500/10 hover:bg-red-500/20 text-red-400 transition-colors flex items-center gap-1">
                            <Trash size={12} /> Delete ({selectedIds.size})
                        </button>
                    </div>
                </div>
            )}

            {/* Download Detail Panel */}
            {detailId && !selectMode && (() => {
                const detailItem = data.find(d => d.id === detailId);
                return detailItem ? <DownloadDetailPanel item={detailItem} onClose={() => setDetailId(null)} /> : null;
            })()}

            {/* Delete Confirmation Modal */}
            {deleteConfirmId && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
                    <div className="bg-th-surface border border-th-overlay rounded-xl p-6 w-[400px] shadow-2xl">
                        <h3 className="text-xl font-bold text-th-text mb-2">Delete Download</h3>
                        <p className="text-th-text-s mb-6">Are you sure you want to delete this task?</p>
                        <div className="flex items-center gap-2 mb-6 p-3 bg-th-raised rounded-lg border border-th-border">
                            <Checkbox id="deleteFile" checked={deleteFile}
                                onChange={setDeleteFile} />
                            <label htmlFor="deleteFile" className="text-sm text-th-text-s select-none cursor-pointer">
                                Delete file from disk also?
                            </label>
                        </div>
                        <div className="flex justify-end gap-3">
                            <button onClick={() => setDeleteConfirmId(null)}
                                className="px-4 py-2 rounded-lg text-th-text-s hover:bg-th-raised hover:text-th-text transition-colors bg-transparent">Cancel</button>
                            <button onClick={() => handleDelete(deleteConfirmId, deleteFile)}
                                className="px-4 py-2 rounded-lg bg-red-600 hover:bg-red-500 text-white font-medium transition-colors shadow-lg shadow-red-900/20">Delete</button>
                        </div>
                    </div>
                </div>
            )}

            {/* Bulk Delete Modal */}
            {bulkDeleteConfirm && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
                    <div className="bg-th-surface border border-th-overlay rounded-xl p-6 w-[400px] shadow-2xl">
                        <h3 className="text-xl font-bold text-th-text mb-2">Delete {selectedIds.size} Downloads</h3>
                        <p className="text-th-text-s mb-6">This will remove {selectedIds.size} download(s) from history.</p>
                        <div className="flex items-center gap-2 mb-6 p-3 bg-th-raised rounded-lg border border-th-border">
                            <Checkbox id="bulkDeleteFile" checked={bulkDeleteFile}
                                onChange={setBulkDeleteFile} />
                            <label htmlFor="bulkDeleteFile" className="text-sm text-th-text-s select-none cursor-pointer">
                                Also delete files from disk
                            </label>
                        </div>
                        <div className="flex justify-end gap-3">
                            <button onClick={() => { setBulkDeleteConfirm(false); setBulkDeleteFile(false); }}
                                className="px-4 py-2 rounded-lg text-th-text-s hover:bg-th-raised hover:text-th-text transition-colors bg-transparent">Cancel</button>
                            <button onClick={handleBulkDelete}
                                className="px-4 py-2 rounded-lg bg-red-600 hover:bg-red-500 text-white font-medium transition-colors shadow-lg shadow-red-900/20">Delete {selectedIds.size} Items</button>
                        </div>
                    </div>
                </div>
            )}

            <ContextMenu
                x={contextMenu.x}
                y={contextMenu.y}
                visible={contextMenu.id !== null}
                onClose={() => setContextMenu({ ...contextMenu, id: null })}
                status={contextItem?.status || ""}
                onOpen={() => contextMenu.id && onOpenFile(contextMenu.id)}
                onShowInFolder={() => contextMenu.id && onOpenFolder(contextMenu.id)}
                onPause={() => contextMenu.id && handlePause(contextMenu.id)}
                onResume={() => contextMenu.id && handleResume(contextMenu.id)}
                onStop={() => contextMenu.id && handleStop(contextMenu.id)}
                onCopyLink={() => { if (contextItem) handleCopyLink(contextItem.url); }}
                onDelete={() => contextMenu.id && setDeleteConfirmId(contextMenu.id)}
                onRetry={() => { if (contextItem) handleRetry(contextItem.url); }}
                onSetPriority={(p) => contextMenu.id && onSetPriority(contextMenu.id, p)}
            />
        </>
    );
};
