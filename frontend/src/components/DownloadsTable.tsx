import React, { useState, useMemo } from 'react';
import {
    useReactTable,
    getCoreRowModel,
    getSortedRowModel,
    flexRender,
    createColumnHelper,
    SortingState
} from '@tanstack/react-table';
import { DownloadItem } from '../types';
import { File, FileVideo, FileArchive, CheckCircle, AlertCircle, Pause, Play, Trash, Folder } from 'lucide-react';
import prettyBytes from 'pretty-bytes';
import { cn } from '../utils';
import { ContextMenu } from './ContextMenu';

interface DownloadsTableProps {
    data: DownloadItem[];
    onOpenFile: (id: string) => void;
    onOpenFolder: (id: string) => void;
}

const columnHelper = createColumnHelper<DownloadItem>();

export const DownloadsTable: React.FC<DownloadsTableProps> = ({ data, onOpenFile, onOpenFolder }) => {
    const [sorting, setSorting] = useState<SortingState>([]);
    const [contextMenu, setContextMenu] = useState<{ x: number, y: number, id: string | null }>({ x: 0, y: 0, id: null });

    const columns = useMemo(() => [
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
                const colors: any = {
                    downloading: "bg-blue-500/10 text-blue-400 border-blue-500/20",
                    completed: "bg-green-500/10 text-green-400 border-green-500/20",
                    error: "bg-red-500/10 text-red-400 border-red-500/20",
                    paused: "bg-yellow-500/10 text-yellow-400 border-yellow-500/20",
                    pending: "bg-slate-500/10 text-slate-400 border-slate-500/20"
                };
                return (
                    <span className={cn(
                        "px-2.5 py-1 rounded-md text-xs font-semibold border capitalize flex w-fit items-center gap-1.5",
                        colors[s] || colors.pending
                    )}>
                        {s === 'downloading' && <span className="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse"></span>}
                        {s}
                    </span>
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
            cell: info => (
                <div className="flex flex-col">
                    <span className="text-slate-300 font-medium text-sm">{prettyBytes(info.getValue() || 0)}</span>
                    <span className="text-slate-500 text-xs font-mono">{info.row.original.eta || '--'}</span>
                </div>
            )
        }),
        columnHelper.display({
            id: 'actions',
            header: '',
            cell: info => (
                <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button className="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors" title="Pause">
                        <Pause size={16} />
                    </button>
                    <button className="p-2 hover:bg-slate-700 rounded-lg text-slate-400 hover:text-white transition-colors" onClick={() => onOpenFolder(info.row.original.id)} title="Folder">
                        <Folder size={16} />
                    </button>
                    <button className="p-2 hover:bg-red-900/50 rounded-lg text-slate-400 hover:text-red-400 transition-colors" title="Delete">
                        <Trash size={16} />
                    </button>
                </div>
            )
        })
    ], []);

    const table = useReactTable({
        data,
        columns,
        state: { sorting },
        onSortingChange: setSorting,
        getCoreRowModel: getCoreRowModel(),
        getSortedRowModel: getSortedRowModel(),
    });

    const handleContextMenu = (e: React.MouseEvent, id: string) => {
        e.preventDefault();
        setContextMenu({ x: e.clientX, y: e.clientY, id });
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
                                onDoubleClick={() => onOpenFile(row.original.id)}
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

            <ContextMenu
                x={contextMenu.x}
                y={contextMenu.y}
                visible={contextMenu.id !== null}
                onClose={() => setContextMenu({ ...contextMenu, id: null })}
                status=""
                onOpen={() => contextMenu.id && onOpenFile(contextMenu.id)}
                onShowInFolder={() => contextMenu.id && onOpenFolder(contextMenu.id)}
                onCopyLink={() => { }}
                onDelete={() => { }}
            />
        </>
    );
};

