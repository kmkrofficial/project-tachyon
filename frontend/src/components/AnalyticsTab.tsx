import React, { useState } from 'react';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { Activity, Download, HardDrive, Wifi, Zap } from 'lucide-react';
import { useTachyon } from '../hooks/useTachyon';
import prettyBytes from 'pretty-bytes';
import { cn } from '../utils';

export const AnalyticsTab = () => {
    const { runSpeedTest, getLifetimeStats } = useTachyon();
    const [speedTestResult, setSpeedTestResult] = useState<{ dl: number, ul: number, ping: number } | null>(null);
    const [testing, setTesting] = useState(false);
    const [stats, setStats] = useState({ totalBytes: 0, files: 0 });

    React.useEffect(() => {
        getLifetimeStats().then(bytes => {
            setStats({ totalBytes: bytes, files: Math.floor(bytes / 1024 / 1024 / 50) });
        });
    }, []);

    const handleTest = async () => {
        setTesting(true);
        try {
            const res = await runSpeedTest();
            if (res) {
                setSpeedTestResult({
                    dl: res.download_mbps,
                    ul: res.upload_mbps,
                    ping: res.ping_ms
                });
            }
        } catch (e) {
            console.error("Speed test failed", e);
        }
        setTesting(false);
    };

    // Mock Data
    const data = [
        { name: 'Mon', downloads: 400 },
        { name: 'Tue', downloads: 300 },
        { name: 'Wed', downloads: 200 },
        { name: 'Thu', downloads: 278 },
        { name: 'Fri', downloads: 189 },
        { name: 'Sat', downloads: 239 },
        { name: 'Sun', downloads: 349 },
    ];

    const fileTypeData = [
        { name: 'Video', value: 400 },
        { name: 'Music', value: 300 },
        { name: 'Docs', value: 300 },
        { name: 'Archives', value: 200 },
    ];
    const COLORS = ['#06b6d4', '#22c55e', '#f59e0b', '#8b5cf6'];

    return (
        <div className="animate-fade-in space-y-6">
            <h2 className="text-2xl font-bold text-white mb-6 hidden">Network Intelligence</h2> {/* Hidden as header exists */}

            {/* 2x2 Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 h-[calc(100vh-140px)]">

                {/* 1. Network Activity (Top Left) */}
                <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 flex flex-col">
                    <div className="flex justify-between items-center mb-6">
                        <h3 className="text-lg font-semibold text-slate-200">Network Activity</h3>
                        <div className="flex gap-2">
                            <span className="px-2 py-1 bg-slate-800 rounded text-xs text-slate-400">24H</span>
                            <span className="px-2 py-1 bg-cyan-900/20 text-cyan-400 rounded text-xs font-bold">LIVE</span>
                        </div>
                    </div>
                    <div className="flex-1 w-full min-h-0">
                        <ResponsiveContainer width="100%" height="100%">
                            <AreaChart data={data}>
                                <defs>
                                    <linearGradient id="colorDl" x1="0" y1="0" x2="0" y2="1">
                                        <stop offset="5%" stopColor="#06b6d4" stopOpacity={0.3} /> {/* Cyan-500 */}
                                        <stop offset="95%" stopColor="#06b6d4" stopOpacity={0} />
                                    </linearGradient>
                                </defs>
                                <XAxis dataKey="name" stroke="#475569" fontSize={12} tickLine={false} axisLine={false} />
                                <YAxis stroke="#475569" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(value) => `${value} MB`} />
                                <Tooltip
                                    contentStyle={{ backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: '8px', color: '#f1f5f9' }}
                                    itemStyle={{ color: '#22d3ee' }}
                                />
                                <Area type="monotone" dataKey="downloads" stroke="#06b6d4" strokeWidth={3} fillOpacity={1} fill="url(#colorDl)" />
                            </AreaChart>
                        </ResponsiveContainer>
                    </div>
                </div>

                {/* 2. Speed Test (Top Right) */}
                <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 relative overflow-hidden group">
                    <div className="absolute top-0 right-0 p-32 bg-cyan-500/5 blur-[100px] rounded-full pointer-events-none"></div>

                    <h3 className="text-lg font-semibold text-slate-200 mb-8 relative z-10">Connection Quality</h3>

                    <div className="flex flex-col items-center justify-center h-[60%] relative z-10">
                        {/* Gauge UI Simulation */}
                        <div className="w-48 h-24 bg-slate-800 rounded-t-full relative overflow-hidden mb-6">
                            <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-40 h-20 bg-slate-900 rounded-t-full flex items-end justify-center pb-2">
                                <span className={cn("text-3xl font-mono font-bold transition-all", testing ? "text-cyan-400 animate-pulse" : "text-white")}>
                                    {speedTestResult ? speedTestResult.dl.toFixed(0) : "--"}
                                </span>
                                <span className="text-xs text-slate-500 mb-1 ml-1">Mbps</span>
                            </div>
                            <div className="absolute bottom-0 left-0 w-full h-2 bg-slate-700"></div>
                            {/* Needle Animation would go here */}
                        </div>

                        <button
                            onClick={handleTest}
                            disabled={testing}
                            className="bg-cyan-600 hover:bg-cyan-500 text-white px-8 py-3 rounded-full font-bold shadow-lg shadow-cyan-500/20 active:scale-95 transition-all disabled:opacity-50"
                        >
                            {testing ? "Testing..." : "Run Speed Test"}
                        </button>
                    </div>

                    <div className="grid grid-cols-3 gap-4 mt-auto pt-6 border-t border-slate-800 relative z-10">
                        <div className="text-center">
                            <p className="text-xs text-slate-500 uppercase">Ping</p>
                            <p className="text-lg font-mono text-yellow-400">{speedTestResult ? speedTestResult.ping : '-- '} ms</p>
                        </div>
                        <div className="text-center border-l border-slate-800">
                            <p className="text-xs text-slate-500 uppercase">Download</p>
                            <p className="text-lg font-mono text-cyan-400">{speedTestResult ? speedTestResult.dl.toFixed(1) : '-- '} <span className="text-xs">Mb</span></p>
                        </div>
                        <div className="text-center border-l border-slate-800">
                            <p className="text-xs text-slate-500 uppercase">Upload</p>
                            <p className="text-lg font-mono text-blue-400">{speedTestResult ? speedTestResult.ul.toFixed(1) : '-- '} <span className="text-xs">Mb</span></p>
                        </div>
                    </div>
                </div>

                {/* 3. Storage Breakdown (Bottom Left - merged with Stats) */}
                <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6">
                    <h3 className="text-lg font-semibold text-slate-200 mb-4">Library Composition</h3>
                    <div className="flex items-center h-[200px]">
                        <ResponsiveContainer width="50%" height="100%">
                            <PieChart>
                                <Pie
                                    data={fileTypeData}
                                    innerRadius={50}
                                    outerRadius={70}
                                    paddingAngle={5}
                                    dataKey="value"
                                    stroke="none"
                                >
                                    {fileTypeData.map((entry, index) => (
                                        <Cell key={`cell - ${index} `} fill={COLORS[index % COLORS.length]} />
                                    ))}
                                </Pie>
                            </PieChart>
                        </ResponsiveContainer>
                        <div className="w-[50%] space-y-3">
                            {fileTypeData.map((entry, index) => (
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
                </div>

                {/* 4. Lifetime Stats (Bottom Right) */}
                <div className="bg-slate-900 border border-slate-800 rounded-2xl p-6 grid grid-cols-2 gap-4">
                    <StatBox label="Lifetime Download" value={prettyBytes(stats.totalBytes)} />
                    <StatBox label="Files Processed" value={stats.files.toString()} />
                    <StatBox label="Peak Speed" value="48.2 MB/s" />
                    <StatBox label="Active Time" value="142h" />
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
