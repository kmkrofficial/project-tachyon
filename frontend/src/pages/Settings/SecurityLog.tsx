import React, { useEffect, useState, useRef } from 'react';
import { Shield, RefreshCw, Scan, CheckCircle, AlertTriangle, XCircle } from 'lucide-react';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';

interface LogEntry {
    id: string;
    timestamp: string;
    source_ip: string;
    user_agent: string;
    action: string;
    status: number;
    details: string;
}

interface ScanResult {
    file: string;
    status: string; // "clean", "threat", "error"
    threat_name?: string;
    timestamp: string;
}

export const SecurityLog: React.FC = () => {
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [scanResults, setScanResults] = useState<ScanResult[]>([]);
    const [polling, setPolling] = useState(true);
    const [activeTab, setActiveTab] = useState<'audit' | 'scans'>('audit');
    const scrollRef = useRef<HTMLDivElement>(null);

    const fetchLogs = async () => {
        if (window.go?.main?.App?.GetRecentAuditLogs) {
            try {
                const data = await window.go.main.App.GetRecentAuditLogs();
                setLogs(data || []);
            } catch (err) {
                console.error("Failed to fetch audit logs", err);
            }
        }
    };

    useEffect(() => {
        fetchLogs();
        if (!polling) return;

        const interval = setInterval(fetchLogs, 2000);
        return () => clearInterval(interval);
    }, [polling]);

    // Listen for security scan result events
    useEffect(() => {
        const handleScanResult = (result: ScanResult) => {
            setScanResults(prev => [{
                ...result,
                timestamp: new Date().toISOString()
            }, ...prev.slice(0, 99)]); // Keep last 100
        };

        EventsOn("security:scan_result", handleScanResult);
        return () => {
            EventsOff("security:scan_result");
        };
    }, []);

    // Auto-scroll to bottom on new logs
    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
        }
    }, [logs, scanResults]);

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'clean': return <CheckCircle size={14} className="text-green-500" />;
            case 'threat': return <AlertTriangle size={14} className="text-red-500" />;
            case 'error': return <XCircle size={14} className="text-yellow-500" />;
            default: return <Scan size={14} className="text-gray-500" />;
        }
    };

    return (
        <div className="space-y-4 h-full flex flex-col">
            <div className="flex justify-between items-center">
                <div className="flex items-center gap-2">
                    <Shield className="text-cyan-500" size={20} />
                    <h3 className="text-lg font-medium text-white">Security Dashboard</h3>
                </div>
                <div className="flex items-center gap-2">
                    {/* Tab buttons */}
                    <button
                        onClick={() => setActiveTab('audit')}
                        className={`px-3 py-1 rounded-lg text-sm transition-colors ${activeTab === 'audit' ? 'bg-cyan-900/50 text-cyan-400' : 'bg-gray-800 text-gray-400'}`}
                    >
                        Audit Log
                    </button>
                    <button
                        onClick={() => setActiveTab('scans')}
                        className={`px-3 py-1 rounded-lg text-sm transition-colors ${activeTab === 'scans' ? 'bg-cyan-900/50 text-cyan-400' : 'bg-gray-800 text-gray-400'}`}
                    >
                        Scan Results
                    </button>
                    <button
                        onClick={() => setPolling(!polling)}
                        className={`p-2 rounded-lg transition-colors ${polling ? 'bg-cyan-900/30 text-cyan-400' : 'bg-gray-800 text-gray-400'}`}
                        title={polling ? "Polling Active" : "Polling Paused"}
                    >
                        <RefreshCw size={16} className={polling ? "animate-spin-slow" : ""} />
                    </button>
                </div>
            </div>

            <div className="bg-black/50 border border-gray-800 rounded-lg flex-1 overflow-hidden flex flex-col font-mono text-xs">
                {activeTab === 'audit' ? (
                    <>
                        <div className="flex bg-gray-900/80 p-2 border-b border-gray-800 text-gray-500 font-bold uppercase tracking-wider">
                            <div className="w-24">Time</div>
                            <div className="w-24">IP</div>
                            <div className="w-16">Status</div>
                            <div className="flex-1">Action</div>
                        </div>

                        <div ref={scrollRef} className="overflow-y-auto flex-1 p-2 space-y-1 scrollbar-thin scrollbar-thumb-gray-700">
                            {logs.length === 0 ? (
                                <div className="text-gray-600 text-center py-8 italic">No audit logs available</div>
                            ) : (
                                logs.map((log) => (
                                    <div key={log.id} className="flex gap-2 hover:bg-white/5 p-1 rounded transition-colors group">
                                        <div className="w-24 text-gray-500 shrink-0">
                                            {new Date(log.timestamp).toLocaleTimeString()}
                                        </div>
                                        <div className="w-24 text-blue-400 shrink-0 truncate" title={log.source_ip}>
                                            {log.source_ip}
                                        </div>
                                        <div className={`w-16 font-bold shrink-0 ${log.status >= 400 ? 'text-red-500' :
                                            log.status >= 300 ? 'text-yellow-500' : 'text-green-500'
                                            }`}>
                                            {log.status}
                                        </div>
                                        <div className="flex-1 text-gray-300 break-all">
                                            <span className="text-purple-400 mr-2">[{log.action}]</span>
                                            <span className="text-gray-400">{log.details}</span>
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>
                    </>
                ) : (
                    <>
                        <div className="flex bg-gray-900/80 p-2 border-b border-gray-800 text-gray-500 font-bold uppercase tracking-wider">
                            <div className="w-20">Time</div>
                            <div className="w-16">Status</div>
                            <div className="flex-1">File</div>
                            <div className="w-32">Threat</div>
                        </div>

                        <div className="overflow-y-auto flex-1 p-2 space-y-1 scrollbar-thin scrollbar-thumb-gray-700">
                            {scanResults.length === 0 ? (
                                <div className="text-gray-600 text-center py-8 italic">No scan results yet. Download files to see scan results.</div>
                            ) : (
                                scanResults.map((scan, idx) => (
                                    <div key={idx} className="flex gap-2 hover:bg-white/5 p-1 rounded transition-colors group items-center">
                                        <div className="w-20 text-gray-500 shrink-0">
                                            {new Date(scan.timestamp).toLocaleTimeString()}
                                        </div>
                                        <div className="w-16 shrink-0 flex items-center gap-1">
                                            {getStatusIcon(scan.status)}
                                            <span className={`text-xs ${scan.status === 'clean' ? 'text-green-500' :
                                                    scan.status === 'threat' ? 'text-red-500' : 'text-yellow-500'
                                                }`}>
                                                {scan.status}
                                            </span>
                                        </div>
                                        <div className="flex-1 text-gray-300 truncate" title={scan.file}>
                                            {scan.file.split(/[/\\]/).pop()}
                                        </div>
                                        <div className="w-32 text-red-400 truncate">
                                            {scan.threat_name || '-'}
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>
                    </>
                )}
            </div>
        </div>
    );
};
