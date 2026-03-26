import React, { useState, useEffect } from 'react';
import { Download, Upload, Server, Clock, Wifi } from 'lucide-react';
// @ts-ignore
import { RunNetworkSpeedTest, GetSpeedTestHistory, ClearSpeedTestHistory } from '../../wailsjs/go/app/App';
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

    const phaseProgress = livePhase?.phase === 'connecting' ? 0
        : livePhase?.phase === 'ping' ? 1
        : livePhase?.phase === 'download' ? 2
        : livePhase?.phase === 'upload' ? 3 : -1;

    return (
        <div className="space-y-4 animate-fade-in">
            {/* Test Card */}
            <div className="bg-th-surface border border-th-border rounded-xl overflow-hidden">
                {/* Top bar: Go button + status */}
                <div className="flex items-center justify-between px-5 py-3 border-b border-th-border">
                    <div className="flex items-center gap-3 text-sm text-th-text-s">
                        {displayServer && (
                            <span className="flex items-center gap-1.5">
                                <Server size={13} className="text-th-text-m" /> {displayServer}
                            </span>
                        )}
                        {displayISP && (
                            <span className="flex items-center gap-1.5">
                                <Wifi size={13} className="text-th-text-m" /> {displayISP}
                            </span>
                        )}
                        {!displayServer && !displayISP && (
                            <span className="text-th-text-m text-xs">Press Go to start a speed test</span>
                        )}
                    </div>
                    <button
                        onClick={runTest}
                        disabled={isRunning}
                        className={`px-5 py-1.5 rounded-lg font-bold text-sm uppercase tracking-wider transition-all ${isRunning
                            ? 'bg-th-accent/20 text-th-accent-t/60 cursor-not-allowed'
                            : 'bg-th-accent hover:bg-th-accent-h text-white shadow-lg shadow-th-accent/30 active:scale-95'
                        }`}
                    >
                        {isRunning ? (phaseLabels[livePhase?.phase ?? ''] || 'Testing...') : 'Go'}
                    </button>
                </div>

                {/* Download / Ping+Jitter / Upload */}
                <div className="grid grid-cols-[1fr_auto_1fr] divide-x divide-th-border">
                    <div className={`py-6 text-center transition-all ${livePhase?.phase === 'download' ? 'bg-green-500/5' : ''}`}>
                        <div className="flex items-center justify-center gap-1.5 mb-1">
                            <Download size={14} className="text-green-400" />
                            <span className="text-xs text-th-text-m uppercase tracking-wider">Download</span>
                        </div>
                        <div className="text-4xl font-bold text-th-text tabular-nums">
                            {displayDownload !== null ? displayDownload.toFixed(1) : <span className="text-th-text-m">--</span>}
                        </div>
                        <div className="text-xs text-th-text-m mt-0.5">Mbps</div>
                    </div>
                    <div className="flex flex-col items-center justify-center gap-3 px-6">
                        <div className={`text-center transition-colors ${livePhase?.phase === 'ping' ? 'text-yellow-300' : 'text-th-text-s'}`}>
                            <div className="text-[10px] text-th-text-m uppercase tracking-wider mb-0.5">Ping</div>
                            <div className="text-lg font-semibold text-th-text tabular-nums leading-tight">{displayPing ?? '--'}<span className="text-[10px] font-normal text-th-text-m ml-0.5">ms</span></div>
                        </div>
                        <div className="text-center text-th-text-s">
                            <div className="text-[10px] text-th-text-m uppercase tracking-wider mb-0.5">Jitter</div>
                            <div className="text-lg font-semibold text-th-text tabular-nums leading-tight">{result?.jitter_ms ?? '--'}<span className="text-[10px] font-normal text-th-text-m ml-0.5">ms</span></div>
                        </div>
                    </div>
                    <div className={`py-6 text-center transition-all ${livePhase?.phase === 'upload' ? 'bg-purple-500/5' : ''}`}>
                        <div className="flex items-center justify-center gap-1.5 mb-1">
                            <Upload size={14} className="text-purple-400" />
                            <span className="text-xs text-th-text-m uppercase tracking-wider">Upload</span>
                        </div>
                        <div className="text-4xl font-bold text-th-text tabular-nums">
                            {displayUpload !== null ? displayUpload.toFixed(1) : <span className="text-th-text-m">--</span>}
                        </div>
                        <div className="text-xs text-th-text-m mt-0.5">Mbps</div>
                    </div>
                </div>

                {/* Progress */}
                <div className="border-t border-th-border px-5 py-5 mt-1">

                    {/* Progress Steps (always visible) */}
                    <div className="flex gap-1">
                        {['Server', 'Ping', 'Download', 'Upload'].map((step, i) => (
                            <div key={step} className="flex-1 flex flex-col items-center gap-1">
                                <div className={`h-1 w-full rounded-full transition-all duration-500 ${
                                    isRunning && i < phaseProgress ? 'bg-th-accent'
                                    : isRunning && i === phaseProgress ? 'bg-th-accent animate-pulse'
                                    : !isRunning && result ? 'bg-th-accent/40'
                                    : 'bg-th-raised'
                                }`} />
                                <span className={`text-[10px] ${
                                    isRunning && i <= phaseProgress ? 'text-th-accent-t'
                                    : !isRunning && result ? 'text-th-text-s'
                                    : 'text-th-text-m'
                                }`}>{step}</span>
                            </div>
                        ))}
                    </div>
                </div>

                {error && (
                    <div className="border-t border-red-900/50 px-5 py-3 bg-red-900/10 text-red-300 text-sm text-center">
                        {error}
                    </div>
                )}
            </div>

            {/* History Table */}
            <div className="bg-th-surface border border-th-border rounded-xl overflow-hidden">
                <div className="px-4 py-2.5 border-b border-th-border flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <Clock size={14} className="text-th-text-s" />
                        <span className="text-sm font-semibold text-th-text">History</span>
                    </div>
                    {history.length > 0 && (
                        <button
                            onClick={async () => { await ClearSpeedTestHistory(); setHistory([]); }}
                            className="text-xs text-th-text-m hover:text-red-400 transition-colors"
                        >
                            Clear All
                        </button>
                    )}
                </div>
                <table className="w-full text-left text-sm text-th-text-s">
                    <thead className="bg-th-base text-th-text font-medium uppercase text-xs tracking-wider">
                        <tr>
                            <th className="px-4 py-2">Date</th>
                            <th className="px-4 py-2">Download</th>
                            <th className="px-4 py-2">Upload</th>
                            <th className="px-4 py-2">Ping</th>
                            <th className="px-4 py-2">Jitter</th>
                            <th className="px-4 py-2">ISP</th>
                        </tr>
                    </thead>
                    <tbody className="divide-y divide-th-border">
                        {history.length === 0 ? (
                            <tr>
                                <td colSpan={6} className="p-6 text-center text-th-text-m italic text-sm">
                                    No speed tests run yet.
                                </td>
                            </tr>
                        ) : (
                            history.map((row, i) => (
                                <tr key={i} className="hover:bg-th-raised/50 transition-colors">
                                    <td className="px-4 py-2 font-mono text-xs">{new Date(row.timestamp).toLocaleString()}</td>
                                    <td className="px-4 py-2 text-green-400 font-bold tabular-nums">{row.download_mbps.toFixed(1)} Mbps</td>
                                    <td className="px-4 py-2 text-purple-400 font-bold tabular-nums">{row.upload_mbps.toFixed(1)} Mbps</td>
                                    <td className="px-4 py-2 tabular-nums">{row.ping_ms} ms</td>
                                    <td className="px-4 py-2 tabular-nums">{row.jitter_ms} ms</td>
                                    <td className="px-4 py-2">{row.isp}</td>
                                </tr>
                            ))
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

