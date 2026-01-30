import React, { useState, useMemo, memo, useCallback } from 'react';
import {
    useReactTable,
    getCoreRowModel,
    getSortedRowModel,
    flexRender,
    createColumnHelper,
    SortingState
} from '@tanstack/react-table';
import { DownloadItem } from '../types';
import { File, FileVideo, FileArchive, CheckCircle, AlertCircle, Pause, Play, Trash, Folder, Square, RotateCcw, ChevronUp, ChevronDown, ChevronsUp, ChevronsDown, AlertTriangle } from 'lucide-react';
import prettyBytes from 'pretty-bytes';
import { cn } from '../utils';
import { ContextMenu } from './ContextMenu';

interface DownloadsTableProps {
    data: DownloadItem[];
    onOpenFile: (id: string) => void;
    onOpenFolder: (id: string) => void;
    onReorder: (id: string, direction: string) => void;
    onSetPriority: (id: string, priority: number) => void;
    addToast: (type: 'success' | 'error' | 'warning' | 'info', title: string, message: string) => void;
}

// Memoized action buttons - only re-renders when status/id/url change, not on progress updates
interface RowActionsProps {
    id: string;
    url: string;
    status: string;
    onPause: (id: string) => void;
    onResume: (id: string) => void;
    onStop: (id: string) => void;
    onRetry: (url: string) => void;
    onOpenFolder: (id: string) => void;
    onReorder: (id: string, direction: string) => void;
    onDelete: (id: string) => void;
    fileExists?: boolean;
}

const RowActions = memo(({ id, url, status, onPause, onResume, onStop, onRetry, onOpenFolder, onReorder, onDelete, fileExists }: RowActionsProps) => {
    const isCompleted = status === 'completed';
    const isDownloading = status === 'downloading';
    const isPaused = status === 'paused';
    const isPending = status === 'pending';
    const isError = status === 'error';
    const isStopped = status === 'stopped';

    return (
        <div className="flex items-center justify-end gap-1 opacity-100 transition-opacity">
            {isPending && (
                <div className="flex gap-0.5 mr-2 bg-slate-800 rounded p-0.5">
                    <button onClick={() => onReorder(id, "first")} className="p-1 hover:text-white text-slate-400 hover:bg-slate-700 rounded" title="Move to Top"><ChevronsUp size={14} /></button>
                    <button onClick={() => onReorder(id, "prev")} className="p-1 hover:text-white text-slate-400 hover:bg-slate-700 rounded" title="Move Up"><ChevronUp size={14} /></button>
                    <button onClick={() => onReorder(id, "next")} className="p-1 hover:text-white text-slate-400 hover:bg-slate-700 rounded" title="Move Down"><ChevronDown size={14} /></button>
                    <button onClick={() => onReorder(id, "last")} className="p-1 hover:text-white text-slate-400 hover:bg-slate-700 rounded" title="Move to Bottom"><ChevronsDown size={14} /></button>
                </div>
            )}

            {(isDownloading || isPaused || isPending) && (
                <>
                    <button
                        className="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors"
                        title={isPaused ? "Resume" : "Pause"}
                        onClick={() => isPaused ? onResume(id) : onPause(id)}
                    >
                        {isPaused ? <Play size={16} /> : <Pause size={16} />}
                    </button>
                    <button
                        className="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors"
                        title="Stop / Cancel"
                        onClick={() => onStop(id)}
                    >
                        <Square size={16} fill="currentColor" className="opacity-80" />
                    </button>
                </>
            )}

            {/* Retry button for Error, Stopped, or Completed but Missing File */}
            {(isError || isStopped || (isCompleted && !fileExists)) && (
                <button
                    className="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors"
                    title="Re-download"
                    onClick={() => onRetry(url)}
                >
                    <RotateCcw size={16} />
                </button>
            )}

            {isCompleted && fileExists && (
                <button
                    className="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors"
                    onClick={() => onOpenFolder(id)}
                    title="Show in Folder"
                >
                    <Folder size={16} />
                </button>
            )}

            <button
                className="p-2 hover:bg-red-900/50 rounded-lg text-slate-400 hover:text-red-400 transition-colors"
                title="Delete"
                onClick={() => onDelete(id)}
            >
                <Trash size={16} />
            </button>
        </div>
    );
}, (prevProps, nextProps) => {
    // Custom comparison - only re-render if status, id, or url change
    return prevProps.status === nextProps.status &&
        prevProps.id === nextProps.id &&
        prevProps.url === nextProps.url &&
        prevProps.fileExists === nextProps.fileExists;
});

const columnHelper = createColumnHelper<DownloadItem>();

export const DownloadsTable: React.FC<DownloadsTableProps> = ({ data, onOpenFile, onOpenFolder, onReorder, onSetPriority, addToast }) => {
    const [sorting, setSorting] = useState<SortingState>([]);
    const [contextMenu, setContextMenu] = useState<{ x: number, y: number, id: string | null }>({ x: 0, y: 0, id: null });
    const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
    const [deleteFile, setDeleteFile] = useState(false);

    const handlePause = useCallback(async (id: string) => {
        try {
            await window.go.main.App.PauseDownload(id);
        } catch (e) {
            console.error(e);
        }
    }, []);

    const handleResume = useCallback(async (id: string) => {
        try {
            await window.go.main.App.ResumeDownload(id);
        } catch (e: any) {
            console.error(e);
            addToast('error', 'Resume Failed', typeof e === 'string' ? e : e.message);
        }
    }, [addToast]);

    const handleStop = useCallback(async (id: string) => {
        try {
            await window.go.main.App.StopDownload(id);
        } catch (e) {
            console.error(e);
        }
    }, []);

    const handleRetry = useCallback(async (url: string) => {
        try {
            await window.go.main.App.AddDownload(url);
            addToast('info', 'Re-downloading', 'Added download back to queue');
        } catch (e: any) {
            console.error(e);
            addToast('error', 'Retry Failed', typeof e === 'string' ? e : e.message);
        }
    }, [addToast]);

    const handleDelete = async (id: string, withFile: boolean) => {
        try {
            await window.go.main.App.DeleteDownload(id, withFile);
            // If we are here, it was successful or returned void
            // But if backend throws error, it goes to catch
            setDeleteConfirmId(null);
            if (withFile) {
                // If backend processed it but file delete failed, how do we know?
                // Wails throws the error if the backend returns error.
                // Our backend modification returns formatted error starting with "WARNING:"
            }
        } catch (e: any) {
            console.error(e);
            if (typeof e === 'string' && e.startsWith("WARNING:")) {
                setDeleteConfirmId(null); // It was deleted from DB
                addToast('warning', 'File Deletion Failed', e.replace("WARNING: ", ""));
            } else {
                addToast('error', 'Delete Failed', typeof e === 'string' ? e : e.message);
            }
        }
    };

    const columns = useMemo(() => [
        columnHelper.accessor('queue_order', {
            header: '#',
            cell: info => <span className="text-slate-500 font-mono text-xs">{info.getValue() || '-'}</span>,
            size: 40,
        }),
        columnHelper.accessor('filename', {
            header: 'File Name',
            cell: info => {
                const ext = info.getValue().split('.').pop()?.toLowerCase();
                let Icon = File;
                if (['mp4', 'mkv', 'webm'].includes(ext || '')) Icon = FileVideo;
                if (['zip', 'rar', '7z'].includes(ext || '')) Icon = FileArchive;

                return (
                    <div className="flex items-center gap-4 py-2">
                        <div className="p-3 bg-slate-800 rounded-xl border border-slate-700/50">
                            <Icon size={20} className="text-slate-400 group-hover:text-cyan-400 transition-colors" />
                        </div>
                        <div className="flex flex-col">
                            <span className="font-semibold text-slate-200 group-hover:text-white transition-colors">{info.getValue()}</span>
                            <div className="flex items-center gap-2">
                                <span className="text-[10px] uppercase font-bold text-slate-500 tracking-wider bg-slate-800/50 px-1.5 rounded border border-slate-700/50">
                                    {(ext || 'BIN').toUpperCase()}
                                </span>
                                <span className="text-xs text-slate-500 truncate max-w-[200px]">{info.row.original.url}</span>
                            </div>
                        </div>
                    </div>
                );
            }
        }),
        columnHelper.accessor('status', {
            header: 'Status',
            cell: info => {
                const s = info.getValue();
                const errorMsg = info.row.original.error;
                const fileExists = info.row.original.file_exists !== false; // Default to true if undefined? No, default true is safer to avoid flashing
                // file_exists is boolean | undefined. If undefined (pending/downloading), treat as valid.
                // If status is completed and file_exists is false, show warning.

                const isMissing = s === 'completed' && info.row.original.file_exists === false;

                const colors: any = {
                    downloading: "bg-blue-500/10 text-blue-400 border-blue-500/20",
                    completed: "bg-green-500/10 text-green-400 border-green-500/20",
                    error: "bg-red-500/10 text-red-400 border-red-500/20 cursor-pointer hover:bg-red-500/20",
                    paused: "bg-yellow-500/10 text-yellow-400 border-yellow-500/20",
                    pending: "bg-slate-500/10 text-slate-400 border-slate-500/20"
                };

                const handleErrorClick = () => {
                    if (s === 'error' && errorMsg) {
                        addToast('error', 'Download Error', errorMsg);
                    }
                };

                return (
                    <div className="flex items-center gap-2">
                        <span
                            className={cn(
                                "px-2.5 py-1 rounded-md text-xs font-semibold border capitalize flex w-fit items-center gap-1.5",
                                colors[s] || colors.pending,
                                isMissing ? "opacity-50" : "" // Dim the completed badge if missing
                            )}
                            onClick={s === 'error' ? handleErrorClick : undefined}
                            title={s === 'error' && errorMsg ? `Click to see error: ${errorMsg}` : undefined}
                        >
                            {s === 'downloading' && <span className="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse"></span>}
                            {s === 'error' && <AlertCircle size={12} />}
                            {s}
                        </span>
                        {isMissing && (
                            <span
                                className="px-2 py-0.5 rounded text-[10px] font-bold border border-red-500/30 bg-red-500/10 text-red-400 uppercase tracking-wide flex items-center gap-1"
                                title="File moved or deleted"
                            >
                                <AlertTriangle size={10} />
                                Not Found
                            </span>
                        )}
                    </div>
                );
            }
        }),
        columnHelper.accessor('progress', {
            header: 'Progress',
            cell: info => (
                <div className="w-full max-w-[140px]">
                    <div className="flex justify-between text-xs mb-1.5">
                        <span className="text-slate-400">{info.getValue().toFixed(1)}%</span>
                    </div>
                    {/* Dual Layer Bar */}
                    <div className="h-1.5 w-full bg-slate-800 rounded-full overflow-hidden relative">
                        <div className="absolute inset-0 bg-slate-700/30 w-full" /> {/* Background Track */}
                        <div
                            className={cn(
                                "h-full rounded-full transition-all duration-500 ease-out relative z-10",
                                info.row.original.status === 'completed' ? "bg-green-500" :
                                    info.row.original.status === 'error' ? "bg-red-500" : "bg-cyan-500"
                            )}
                            style={{ width: `${info.getValue()}%` }}
                        >
                            {info.row.original.status === 'downloading' && (
                                <div className="absolute inset-0 bg-white/20 animate-[shimmer_1s_infinite] w-full" style={{ backgroundImage: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.5), transparent)' }}></div>
                            )}
                        </div>
                    </div>
                </div>
            )
        }),
        columnHelper.accessor('speed_MBs', {
            header: 'Speed',
            cell: info => info.row.original.status === 'downloading' ?
                <span className="font-mono text-cyan-400 font-medium">{(info.getValue() || 0).toFixed(1)} MB/s</span> :
                <span className="text-slate-600">-</span>
        }),
        columnHelper.accessor('size', {
            header: 'Size / ETA',
            cell: info => {
                const { status, size, progress, speed_MBs } = info.row.original;
                let etaString = '--';

                if (status === 'downloading') {
                    if (!speed_MBs || speed_MBs <= 0) {
                        etaString = '∞';
                    } else {
                        const remainingBytes = size * (1 - (progress / 100));
                        const speedBytes = speed_MBs * 1024 * 1024;
                        const seconds = remainingBytes / speedBytes;

                        if (seconds < 60) etaString = `${Math.ceil(seconds)}s`;
                        else if (seconds < 3600) etaString = `${Math.floor(seconds / 60)}m ${Math.ceil(seconds % 60)}s`;
                        else etaString = `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
                    }
                } else if (status === 'completed') {
                    etaString = 'Done';
                }

                return (
                    <div className="flex flex-col">
                        <span className="text-slate-300 font-medium text-sm">{prettyBytes(info.getValue() || 0)}</span>
                        <span className="text-slate-500 text-xs font-mono">{etaString}</span>
                    </div>
                );
            }
        }),
        columnHelper.display({
            id: 'actions',
            header: '',
            cell: info => (
                <RowActions
                    id={info.row.original.id}
                    url={info.row.original.url}
                    status={info.row.original.status}
                    onPause={handlePause}
                    onResume={handleResume}
                    onStop={handleStop}
                    onRetry={handleRetry}
                    onOpenFolder={onOpenFolder}
                    onReorder={onReorder}
                    onDelete={(id) => setDeleteConfirmId(id)}
                    fileExists={info.row.original.file_exists}
                />
            )
        })
    ], [handlePause, handleResume, handleStop, handleRetry, onOpenFolder, onReorder]); // Dependencies for memoized handlers

    const table = useReactTable({
        data,
        columns,
        state: { sorting },
        onSortingChange: setSorting,
        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
        getRowId: row => row.id, // STABILIZE ROWS to prevent flickering
    });

    const handleContextMenu = (e: React.MouseEvent, id: string) => {
        e.preventDefault();
        setContextMenu({ x: e.clientX, y: e.clientY, id });
    };

    const handleCopyLink = (url: string) => {
        navigator.clipboard.writeText(url);
        addToast('success', 'Copied', 'Link copied to clipboard');
    };

    return (
        <>
            <div className="w-full">
                <table className="w-full text-left border-collapse">
                    <thead className="bg-slate-900/95 sticky top-0 z-30 backdrop-blur-sm border-b border-slate-800">
                        {table.getHeaderGroups().map(headerGroup => (
                            <tr key={headerGroup.id}>
                                {headerGroup.headers.map(header => (
                                    <th key={header.id} className="px-6 py-4 text-xs font-bold text-slate-500 uppercase tracking-wider cursor-pointer hover:text-slate-300 transition-colors" onClick={header.column.getToggleSortingHandler()}>
                                        {flexRender(header.column.columnDef.header, header.getContext())}
                                        {{
                                            asc: ' ▲',
                                            desc: ' ▼',
                                        }[header.column.getIsSorted() as string] ?? null}
                                    </th>
                                ))}
                            </tr>
                        ))}
                    </thead>
                    <tbody className="divide-y divide-slate-800">
                        {table.getRowModel().rows.map(row => (
                            <tr
                                key={row.id}
                                className="group bg-slate-900 even:bg-slate-800/30 hover:bg-slate-800 transition-colors cursor-default"
                                onContextMenu={(e) => handleContextMenu(e, row.original.id)}
                                onDoubleClick={() => row.original.status === 'completed' && onOpenFile(row.original.id)}
                            >
                                {row.getVisibleCells().map(cell => (
                                    <td key={cell.id} className="px-6 py-3 whitespace-nowrap">
                                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                    </td>
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>

                {data.length === 0 && (
                    <div className="flex flex-col items-center justify-center py-24 text-slate-600">
                        <div className="w-16 h-16 bg-slate-800/50 rounded-full flex items-center justify-center mb-4">
                            <File size={32} className="opacity-50" />
                        </div>
                        <p className="font-medium text-lg">No downloads yet</p>
                        <p className="text-sm">Add a link to get started</p>
                    </div>
                )}
            </div>

            {/* Delete Confirmation Modal */}
            {deleteConfirmId && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
                    <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 w-[400px] shadow-2xl">
                        <h3 className="text-xl font-bold text-slate-200 mb-2">Delete Download</h3>
                        <p className="text-slate-400 mb-6">Are you sure you want to delete this task?</p>

                        <div className="flex items-center gap-2 mb-6 p-3 bg-slate-800 rounded-lg border border-slate-700">
                            <input
                                type="checkbox"
                                id="deleteFile"
                                checked={deleteFile}
                                onChange={e => setDeleteFile(e.target.checked)}
                                className="w-4 h-4 rounded border-gray-600 bg-gray-700 text-cyan-600 focus:ring-cyan-500"
                            />
                            <label htmlFor="deleteFile" className="text-sm text-slate-300 select-none cursor-pointer">
                                Delete file from disk also?
                            </label>
                        </div>

                        <div className="flex justify-end gap-3">
                            <button
                                onClick={() => setDeleteConfirmId(null)}
                                className="px-4 py-2 rounded-lg text-slate-400 hover:bg-slate-800 hover:text-slate-200 transition-colors bg-transparent"
                            >
                                Cancel
                            </button>
                            <button
                                onClick={() => handleDelete(deleteConfirmId, deleteFile)}
                                className="px-4 py-2 rounded-lg bg-red-600 hover:bg-red-500 text-white font-medium transition-colors shadow-lg shadow-red-900/20"
                            >
                                Delete
                            </button>
                        </div>
                    </div>
                </div>
            )}

            <ContextMenu
                x={contextMenu.x}
                y={contextMenu.y}
                visible={contextMenu.id !== null}
                onClose={() => setContextMenu({ ...contextMenu, id: null })}
                status={data.find(d => d.id === contextMenu.id)?.status || ""}
                onOpen={() => contextMenu.id && onOpenFile(contextMenu.id)}
                onShowInFolder={() => contextMenu.id && onOpenFolder(contextMenu.id)}
                onCopyLink={() => {
                    const item = data.find(d => d.id === contextMenu.id);
                    if (item) handleCopyLink(item.url);
                }}
                onDelete={() => contextMenu.id && setDeleteConfirmId(contextMenu.id)}
                onRetry={() => {
                    const item = data.find(d => d.id === contextMenu.id);
                    if (item) handleRetry(item.url);
                }}
                onSetPriority={(p) => contextMenu.id && onSetPriority(contextMenu.id, p)}
            />
        </>
    );
};
