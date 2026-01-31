import React, { useState, useEffect } from 'react';
import { Play, Download, Upload, Activity, Server, Clock, Wifi } from 'lucide-react';
// @ts-ignore
import { RunNetworkSpeedTest, GetSpeedTestHistory } from '../../wailsjs/go/app/App';
// @ts-ignore
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';

interface SpeedTestResult {
    download_mbps: number;
    upload_mbps: number;
    ping_ms: number;
    jitter_ms: number;
    isp: string;
    server_name: string;
    server_location: string;
    timestamp: string;
}

interface SpeedTestPhase {
    phase: string;
    ping_ms?: number;
    download_mbps?: number;
    upload_mbps?: number;
    server_name?: string;
    isp?: string;
    error?: string;
}

const phaseLabels: Record<string, string> = {
    connecting: "Finding server...",
    ping: "Testing latency...",
    download: "Testing download...",
    upload: "Testing upload...",
    complete: "Complete!",
    error: "Test failed"
};

export const SpeedTestTab: React.FC = () => {
    const [isRunning, setIsRunning] = useState(false);
    const [result, setResult] = useState<SpeedTestResult | null>(null);
    const [history, setHistory] = useState<SpeedTestResult[]>([]);
    const [error, setError] = useState("");
    const [livePhase, setLivePhase] = useState<SpeedTestPhase | null>(null);

    const fetchHistory = async () => {
        try {
            const data = await GetSpeedTestHistory();
            if (data && Array.isArray(data)) {
                setHistory(data);
            }
        } catch (e) {
            console.error("Failed to load history", e);
        }
    };

    useEffect(() => {
        fetchHistory();

        // Listen for live phase updates
        const cleanup = EventsOn("speedtest:phase", (data: SpeedTestPhase) => {
            setLivePhase(data);
            if (data.phase === "error") {
                setError(data.error || "Speed test failed");
                setIsRunning(false);
            }
        });

        return () => {
            EventsOff("speedtest:phase");
        };
    }, []);

    const runTest = async () => {
        setIsRunning(true);
        setError("");
        setResult(null);
        setLivePhase({ phase: "connecting" });
        try {
            const res = await RunNetworkSpeedTest();
            if (res) {
                setResult(res);
                fetchHistory(); // Refresh table
            }
        } catch (e: any) {
            setError(typeof e === 'string' ? e : "Speed test failed. Check your connection.");
        } finally {
            setIsRunning(false);
            setLivePhase(null);
        }
    };

    // Get current display values (live or final result)
    const displayPing = livePhase?.ping_ms ?? result?.ping_ms ?? null;
    const displayDownload = livePhase?.download_mbps ?? result?.download_mbps ?? null;
    const displayUpload = livePhase?.upload_mbps ?? result?.upload_mbps ?? null;
    const displayServer = livePhase?.server_name ?? result?.server_name ?? null;
    const displayISP = livePhase?.isp ?? result?.isp ?? null;

    return (
        <div className="space-y-6 animate-fade-in">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold text-white mb-2">Network Speed Test</h1>
                <p className="text-slate-400">Measure your internet connection performance.</p>
            </div>

            {/* Main Test Area */}
            <div className="bg-slate-900 border border-slate-800 rounded-2xl p-8 relative overflow-hidden">
                <div className="absolute top-0 right-0 p-32 bg-cyan-500/10 blur-[100px] rounded-full pointer-events-none" />

                <div className="flex flex-col md:flex-row items-center gap-12 relative z-10">
                    {/* Start Button / Status */}
                    <div className="flex-shrink-0 text-center">
                        <button
                            onClick={runTest}
                            disabled={isRunning}
                            className={`w-48 h-48 rounded-full border-4 flex flex-col items-center justify-center transition-all ${isRunning
                                ? 'border-cyan-500/50 bg-cyan-900/10 animate-pulse cursor-not-allowed'
                                : 'border-cyan-500 hover:border-cyan-400 bg-cyan-500/10 hover:bg-cyan-500/20 hover:scale-105 shadow-[0_0_30px_rgba(6,182,212,0.3)]'
                                }`}
                        >
                            {isRunning ? (
                                <>
                                    <Activity className="text-cyan-400 animate-bounce mb-2" size={48} />
                                    <span className="text-cyan-200 font-mono text-sm tracking-wider uppercase">Testing...</span>
                                </>
                            ) : (
                                <>
                                    <Play className="text-cyan-400 ml-2 mb-2" size={48} />
                                    <span className="text-white font-bold text-lg tracking-wider uppercase">Start</span>
                                </>
                            )}
                        </button>
                        {/* Phase Label */}
                        {isRunning && livePhase && (
                            <div className="mt-4 text-cyan-300 font-medium text-sm">
                                {phaseLabels[livePhase.phase] || livePhase.phase}
                            </div>
                        )}
                    </div>

                    {/* Live/Result Stats */}
                    <div className="flex-1 grid grid-cols-2 md:grid-cols-4 gap-6 w-full">
                        <StatBox
                            icon={<Download className="text-green-400" />}
                            label="Download"
                            value={displayDownload !== null ? displayDownload.toFixed(1) : "--"}
                            unit="Mbps"
                            highlight
                            active={livePhase?.phase === "download"}
                        />
                        <StatBox
                            icon={<Upload className="text-purple-400" />}
                            label="Upload"
                            value={displayUpload !== null ? displayUpload.toFixed(1) : "--"}
                            unit="Mbps"
                            active={livePhase?.phase === "upload"}
                        />
                        <StatBox
                            icon={<Activity className="text-yellow-400" />}
                            label="Ping"
                            value={displayPing !== null ? displayPing.toString() : "--"}
                            unit="ms"
                            active={livePhase?.phase === "ping"}
                        />
                        <StatBox
                            icon={<Wifi className="text-blue-400" />}
                            label="Jitter"
                            value={result ? result.jitter_ms.toString() : "--"}
                            unit="ms"
                        />
                    </div>
                </div>

                {/* Metadata */}
                {(result || isRunning) && (
                    <div className="mt-8 pt-6 border-t border-slate-800 grid grid-cols-1 md:grid-cols-2 gap-4 text-sm text-slate-400">
                        <div className="flex items-center gap-2">
                            <Server size={16} />
                            <span>Server: <span className="text-slate-200">{displayServer || "Finding optimal server..."}</span></span>
                        </div>
                        <div className="flex items-center gap-2">
                            <Wifi size={16} />
                            <span>ISP: <span className="text-slate-200">{displayISP || "--"}</span></span>
                        </div>
                    </div>
                )}

                {error && (
                    <div className="mt-6 p-4 bg-red-900/20 border border-red-900/50 rounded-lg text-red-200 text-center">
                        {error}
                    </div>
                )}
            </div>

            {/* History Table */}
            <div>
                <h2 className="text-xl font-bold text-white mb-4 flex items-center gap-2">
                    <Clock size={20} className="text-slate-400" />
                    History
                </h2>
                <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
                    <table className="w-full text-left text-sm text-slate-400">
                        <thead className="bg-slate-950 text-slate-200 font-medium uppercase text-xs tracking-wider">
                            <tr>
                                <th className="p-4">Date</th>
                                <th className="p-4">Download</th>
                                <th className="p-4">Upload</th>
                                <th className="p-4">Ping</th>
                                <th className="p-4">ISP</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-800">
                            {history.length === 0 ? (
                                <tr>
                                    <td colSpan={5} className="p-8 text-center text-slate-500 italic">
                                        No speed tests run yet.
                                    </td>
                                </tr>
                            ) : (
                                history.map((row, i) => (
                                    <tr key={i} className="hover:bg-slate-800/50 transition-colors">
                                        <td className="p-4 font-mono">{new Date(row.timestamp).toLocaleString()}</td>
                                        <td className="p-4 text-green-400 font-bold">{row.download_mbps.toFixed(1)} Mbps</td>
                                        <td className="p-4 text-purple-400 font-bold">{row.upload_mbps.toFixed(1)} Mbps</td>
                                        <td className="p-4">{row.ping_ms} ms</td>
                                        <td className="p-4">{row.isp}</td>
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    );
};

const StatBox = ({ icon, label, value, unit, highlight, active }: any) => (
    <div className={`bg-slate-950/50 p-4 rounded-xl border flex flex-col items-center justify-center text-center transition-all ${active ? 'border-cyan-500 shadow-[0_0_20px_rgba(6,182,212,0.3)]' : 'border-slate-800'}`}>
        <div className="mb-2 opacity-80">{icon}</div>
        <div className="text-xs text-slate-500 uppercase tracking-wider mb-1">{label}</div>
        <div className={`text-2xl font-bold ${highlight ? 'text-white scale-110' : 'text-slate-200'}`}>
            {value} <span className="text-xs font-normal text-slate-500">{unit}</span>
        </div>
    </div>
);
