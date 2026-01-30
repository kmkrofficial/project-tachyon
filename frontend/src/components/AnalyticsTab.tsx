import React, { useMemo } from 'react';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { Activity, HardDrive, File, Music, Video, Image, Archive } from 'lucide-react';
import { useTachyon } from '../hooks/useTachyon';
import prettyBytes from 'pretty-bytes';

export const AnalyticsTab = () => {
    const { analyticsData, downloads } = useTachyon();

    // Transform Daily History for Chart
    const chartData = useMemo(() => {
        if (!analyticsData || !analyticsData.daily_history) return [];

        // analyticsData.daily_history is { "2024-01-01": 1234 }
        // We need array sorted by date
        const dates = Object.keys(analyticsData.daily_history).sort();
        // If empty, generate last 7 days empty
        if (dates.length === 0) {
            const empty = [];
            for (let i = 6; i >= 0; i--) {
                const d = new Date();
                d.setDate(d.getDate() - i);
                empty.push({ name: d.toLocaleDateString(undefined, { weekday: 'short' }), downloads: 0 });
            }
            return empty;
        }

        return dates.map(date => {
            const bytes = analyticsData.daily_history[date];
            return {
                name: new Date(date).toLocaleDateString(undefined, { weekday: 'short' }), // "Mon"
                fullDate: date,
                downloads: Number((bytes / (1024 * 1024)).toFixed(1)) // MB
            };
        });
    }, [analyticsData]);

    // Calculate Composition
    const compositionData = useMemo(() => {
        const counts: Record<string, number> = { Video: 0, Music: 0, Images: 0, Archives: 0, Docs: 0, Other: 0 };
        Object.values(downloads).forEach(d => {
            const ext = d.filename.split('.').pop()?.toLowerCase();
            if (['mp4', 'mkv', 'avi', 'mov'].includes(ext || '')) counts.Video++;
            else if (['mp3', 'wav', 'flac', 'aac'].includes(ext || '')) counts.Music++;
            else if (['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext || '')) counts.Images++;
            else if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext || '')) counts.Archives++;
            else if (['pdf', 'doc', 'docx', 'txt', 'md'].includes(ext || '')) counts.Docs++;
            else counts.Other++;
        });

        // Filter out zero entries
        return Object.entries(counts)
            .filter(([_, value]) => value > 0)
            .map(([name, value]) => ({ name, value }));
    }, [downloads]);

    const COLORS = ['#06b6d4', '#22c55e', '#f59e0b', '#8b5cf6', '#ec4899', '#64748b'];

    if (!analyticsData) {
        return (
            <div className="flex h-96 items-center justify-center text-slate-500">
                <Activity className="animate-pulse mr-2" /> Loading Analytics...
            </div>
        )
    }

    return (
        <div className="animate-fade-in space-y-6">

            {/* 2 Grid */}
            {/* 2 Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 min-h-[600px] h-[calc(100vh-140px)]">

                {/* 1. Network Activity (Top Left) */}
                <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 flex flex-col">
                    <div className="flex justify-between items-center mb-6">
                        <h3 className="text-lg font-semibold text-slate-200">Network Activity</h3>
                        <div className="flex gap-2">
                            <span className="px-2 py-1 bg-slate-800 rounded text-xs text-slate-400">7 Days</span>
                        </div>
                    </div>
                    <div className="flex-1 w-full min-h-0 relative">
                        {chartData.some(d => d.downloads > 0) ? (
                            <ResponsiveContainer width="100%" height="100%">
                                <AreaChart data={chartData}>
                                    <defs>
                                        <linearGradient id="colorDl" x1="0" y1="0" x2="0" y2="1">
                                            <stop offset="5%" stopColor="#06b6d4" stopOpacity={0.3} />
                                            <stop offset="95%" stopColor="#06b6d4" stopOpacity={0} />
                                        </linearGradient>
                                    </defs>
                                    <XAxis dataKey="name" stroke="#475569" fontSize={12} tickLine={false} axisLine={false} />
                                    <YAxis stroke="#475569" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(value) => `${value} MB`} />
                                    <Tooltip
                                        contentStyle={{ backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: '8px', color: '#f1f5f9' }}
                                        itemStyle={{ color: '#22d3ee' }}
                                        formatter={(value: any) => [`${value} MB`, 'Downloaded']}
                                        labelFormatter={(label) => label}
                                    />
                                    <Area type="monotone" dataKey="downloads" stroke="#06b6d4" strokeWidth={3} fillOpacity={1} fill="url(#colorDl)" />
                                </AreaChart>
                            </ResponsiveContainer>
                        ) : (
                            <div className="absolute inset-0 flex flex-col items-center justify-center text-slate-500">
                                <Activity className="w-12 h-12 mb-3 opacity-20" />
                                <p className="text-sm font-medium text-slate-400">Need more data</p>
                                <p className="text-xs text-slate-600 mt-1">Start downloading to populate the graph</p>
                            </div>
                        )}
                    </div>
                </div>

                {/* Right Column Stack */}
                <div className="flex flex-col gap-6">

                    {/* 2. Library Composition */}
                    <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 flex-1">
                        <h3 className="text-lg font-semibold text-slate-200 mb-4">Library Composition</h3>
                        {compositionData.length === 0 ? (
                            <div className="h-full flex flex-col items-center justify-center text-slate-500 text-sm">
                                <HardDrive size={32} className="mb-2 opacity-50" />
                                No files tracked yet
                            </div>
                        ) : (
                            <div className="flex items-center h-[200px]">
                                <ResponsiveContainer width="50%" height="100%">
                                    <PieChart>
                                        <Pie
                                            data={compositionData}
                                            innerRadius={50}
                                            outerRadius={70}
                                            paddingAngle={5}
                                            dataKey="value"
                                            stroke="none"
                                        >
                                            {compositionData.map((entry, index) => (
                                                <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                                            ))}
                                        </Pie>
                                    </PieChart>
                                </ResponsiveContainer>
                                <div className="w-[50%] space-y-3">
                                    {compositionData.map((entry, index) => (
                                        <div key={entry.name} className="flex items-center justify-between text-sm">
                                            <div className="flex items-center gap-2">
                                                <div className="w-3 h-3 rounded-full" style={{ background: COLORS[index % COLORS.length] }}></div>
                                                <span className="text-slate-400">{entry.name}</span>
                                            </div>
                                            <span className="text-slate-200 font-mono">{entry.value}</span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}
                    </div>

                    {/* 3. Lifetime Stats */}
                    <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 grid grid-cols-2 gap-4">
                        <StatBox label="Lifetime Download" value={prettyBytes(analyticsData.total_downloaded || 0)} />
                        <StatBox label="Files Processed" value={(analyticsData.total_files || 0).toString()} />
                        <StatBox label="Disk Used" value={`${analyticsData.disk_usage?.percent.toFixed(1) || 0}%`} />
                        <StatBox label="Free Space" value={`${analyticsData.disk_usage?.free_gb.toFixed(0) || 0} GB`} />
                    </div>
                </div>
            </div>
        </div>
    );
};

const StatBox = ({ label, value }: any) => (
    <div className="bg-slate-950/50 rounded-xl p-4 flex flex-col justify-center border border-slate-800/50">
        <span className="text-slate-500 text-xs uppercase font-bold tracking-wider mb-1">{label}</span>
        <span className="text-2xl font-bold text-slate-200">{value}</span>
    </div>
);
