import { useState, useEffect, useCallback, useRef } from 'react';
// @ts-ignore
import { RunNetworkSpeedTest, GetSpeedTestHistory, ClearSpeedTestHistory, CancelSpeedTest } from '../../wailsjs/go/app/App';
// @ts-ignore
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';

export interface SpeedTestResult {
    download_mbps: number;
    upload_mbps: number;
    ping_ms: number;
    jitter_ms: number;
    isp: string;
    server_name: string;
    server_location: string;
    timestamp: string;
}

export interface SpeedTestPhase {
    phase: string;
    ping_ms?: number;
    download_mbps?: number;
    upload_mbps?: number;
    server_name?: string;
    isp?: string;
    error?: string;
}

export interface SpeedTestState {
    isRunning: boolean;
    result: SpeedTestResult | null;
    history: SpeedTestResult[];
    error: string;
    livePhase: SpeedTestPhase | null;
    runTest: () => void;
    cancelTest: () => void;
    clearHistory: () => void;
}

export const useSpeedTest = (): SpeedTestState => {
    const [isRunning, setIsRunning] = useState(false);
    const [result, setResult] = useState<SpeedTestResult | null>(null);
    const [history, setHistory] = useState<SpeedTestResult[]>([]);
    const [error, setError] = useState("");
    const [livePhase, setLivePhase] = useState<SpeedTestPhase | null>(null);
    const runningRef = useRef(false);

    const fetchHistory = useCallback(async () => {
        try {
            const data = await GetSpeedTestHistory();
            if (data && Array.isArray(data)) {
                setHistory(data);
            }
        } catch (e) {
            console.error("Failed to load history", e);
        }
    }, []);

    useEffect(() => {
        fetchHistory();

        const cleanup = EventsOn("speedtest:phase", (data: SpeedTestPhase) => {
            if (data.phase === "error") {
                setError(data.error || "Speed test failed");
                setIsRunning(false);
                runningRef.current = false;
                setLivePhase(null);
            } else if (data.phase === "cancelled") {
                setIsRunning(false);
                runningRef.current = false;
                setLivePhase(null);
            } else if (data.phase === "complete") {
                setLivePhase(data);
            } else {
                setLivePhase(data);
            }
        });

        return () => {
            EventsOff("speedtest:phase");
        };
    }, [fetchHistory]);

    const runTest = useCallback(async () => {
        if (runningRef.current) return;
        runningRef.current = true;
        setIsRunning(true);
        setError("");
        setResult(null);
        setLivePhase({ phase: "connecting" });
        try {
            const res = await RunNetworkSpeedTest();
            if (res) {
                setResult(res);
                fetchHistory();
            }
        } catch (e: any) {
            setError(typeof e === 'string' ? e : "Speed test failed. Check your connection.");
        } finally {
            setIsRunning(false);
            runningRef.current = false;
            setLivePhase(null);
        }
    }, [fetchHistory]);

    const cancelTest = useCallback(async () => {
        try {
            await CancelSpeedTest();
        } catch (e) {
            console.error("Failed to cancel speed test", e);
        }
    }, []);

    const clearHistory = useCallback(async () => {
        await ClearSpeedTestHistory();
        setHistory([]);
    }, []);

    return { isRunning, result, history, error, livePhase, runTest, cancelTest, clearHistory };
};
