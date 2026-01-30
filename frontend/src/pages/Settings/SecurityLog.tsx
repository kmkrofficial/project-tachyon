import React, { useEffect, useState, useRef } from 'react';
import { Shield, RefreshCw } from 'lucide-react';

interface LogEntry {
    id: string;
    timestamp: string;
    source_ip: string;
    user_agent: string;
    action: string;
    status: number;
    details: string;
}

export const SecurityLog: React.FC = () => {
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [polling, setPolling] = useState(true);
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

    // Auto-scroll to bottom on new logs
    useEffect(() => {
        if (scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
        }
    }, [logs]);

    return (
        <div className="space-y-4 h-full flex flex-col">
            <div className="flex justify-between items-center">
                <div className="flex items-center gap-2">
                    <Shield className="text-cyan-500" size={20} />
                    <h3 className="text-lg font-medium text-white">Security Audit Log</h3>
                </div>
                <button
                    onClick={() => setPolling(!polling)}
                    className={`p-2 rounded-lg transition-colors ${polling ? 'bg-cyan-900/30 text-cyan-400' : 'bg-gray-800 text-gray-400'}`}
                    title={polling ? "Polling Active" : "Polling Paused"}
                >
                    <RefreshCw size={16} className={polling ? "animate-spin-slow" : ""} />
                </button>
            </div>

            <div className="bg-black/50 border border-gray-800 rounded-lg flex-1 overflow-hidden flex flex-col font-mono text-xs">
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
            </div>
        </div>
    );
};
