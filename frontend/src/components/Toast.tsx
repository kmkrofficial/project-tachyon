import React, { useEffect, useState } from 'react';
import { X, AlertCircle, CheckCircle, Info } from 'lucide-react';
import { cn } from '../utils';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface ToastMessage {
    id: string;
    type: ToastType;
    title: string;
    message: string;
}

interface ToastContainerProps {
    toasts: ToastMessage[];
    removeToast: (id: string) => void;
}

export const ToastContainer: React.FC<ToastContainerProps> = ({ toasts, removeToast }) => {
    return (
        <div className="fixed top-4 right-4 z-[200] flex flex-col gap-3 font-sans">
            {toasts.map(t => (
                <ToastItem key={t.id} toast={t} onDismiss={() => removeToast(t.id)} />
            ))}
        </div>
    );
};

const ToastItem = ({ toast, onDismiss }: { toast: ToastMessage, onDismiss: () => void }) => {
    const [isExiting, setIsExiting] = useState(false);

    useEffect(() => {
        // Start exit animation after 3.5 seconds
        const exitTimer = setTimeout(() => setIsExiting(true), 3500);
        // Actually remove after exit animation completes (500ms animation)
        const removeTimer = setTimeout(onDismiss, 4000);
        return () => {
            clearTimeout(exitTimer);
            clearTimeout(removeTimer);
        };
    }, [onDismiss]);

    const handleDismiss = () => {
        setIsExiting(true);
        setTimeout(onDismiss, 300); // Wait for animation
    };

    const icons = {
        success: <CheckCircle size={20} className="text-green-400" />,
        error: <AlertCircle size={20} className="text-red-400" />,
        warning: <AlertCircle size={20} className="text-yellow-400" />,
        info: <Info size={20} className="text-blue-400" />
    };

    const styles = {
        success: "bg-slate-900 border-green-500/20 shadow-green-900/10",
        error: "bg-slate-900 border-red-500/20 shadow-red-900/10",
        warning: "bg-slate-900 border-yellow-500/20 shadow-yellow-900/10",
        info: "bg-slate-900 border-blue-500/20 shadow-blue-900/10",
    }

    return (
        <div className={cn(
            "w-80 p-4 rounded-xl border shadow-xl flex gap-3 transition-all duration-300 relative overflow-hidden",
            styles[toast.type],
            isExiting ? "opacity-0 translate-x-4" : "opacity-100 translate-x-0 animate-slide-in"
        )}>
            <div className="shrink-0 pt-0.5">{icons[toast.type]}</div>
            <div className="flex-1">
                <h4 className="text-sm font-bold text-slate-200">{toast.title}</h4>
                <p className="text-xs text-slate-400 mt-1 leading-relaxed">{toast.message}</p>
            </div>
            <button onClick={handleDismiss} className="text-slate-500 hover:text-white transition-colors">
                <X size={16} />
            </button>
        </div>
    )
}
