import React, { useState, useRef, useEffect } from 'react';
import { ChevronDown } from 'lucide-react';
import { cn } from '../../utils';

interface DropdownOption {
    value: string;
    label: string;
}

interface DropdownProps {
    value: string;
    onChange: (value: string) => void;
    options: DropdownOption[];
    className?: string;
}

export const Dropdown: React.FC<DropdownProps> = ({ value, onChange, options, className }) => {
    const [open, setOpen] = useState(false);
    const ref = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const handler = (e: MouseEvent) => {
            if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
        };
        document.addEventListener('mousedown', handler);
        return () => document.removeEventListener('mousedown', handler);
    }, []);

    const selected = options.find(o => o.value === value);

    return (
        <div ref={ref} className={cn("relative", className)}>
            <button
                type="button"
                onClick={() => setOpen(prev => !prev)}
                className={cn(
                    "flex items-center gap-1.5 bg-th-surface border rounded-lg px-2.5 py-1.5 text-sm transition-colors w-full",
                    open ? "border-th-accent text-th-text" : "border-th-border text-th-text hover:border-th-overlay"
                )}
            >
                <span className="truncate">{selected?.label ?? value}</span>
                <ChevronDown size={14} className={cn("shrink-0 text-th-text-m transition-transform duration-150", open && "rotate-180")} />
            </button>

            {open && (
                <div className="absolute z-50 mt-1 w-full min-w-[140px] bg-th-surface border border-th-border rounded-lg shadow-xl shadow-black/20 py-1 overflow-hidden">
                    {options.map(opt => (
                        <button
                            key={opt.value}
                            type="button"
                            onClick={() => { onChange(opt.value); setOpen(false); }}
                            className={cn(
                                "w-full text-left px-3 py-1.5 text-sm transition-colors",
                                opt.value === value
                                    ? "bg-th-accent/15 text-th-accent-t"
                                    : "text-th-text-s hover:bg-th-raised hover:text-th-text"
                            )}
                        >
                            {opt.label}
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
};
