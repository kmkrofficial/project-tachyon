import React from 'react';
import { Check, Minus } from 'lucide-react';
import { cn } from '../../utils';

interface CheckboxProps {
    checked: boolean;
    onChange: (checked: boolean) => void;
    indeterminate?: boolean;
    size?: 'sm' | 'md';
    id?: string;
    className?: string;
    onClick?: (e: React.MouseEvent) => void;
}

export const Checkbox: React.FC<CheckboxProps> = ({ checked, onChange, indeterminate, size = 'md', id, className, onClick }) => {
    const dim = size === 'sm' ? 'w-3.5 h-3.5' : 'w-4 h-4';
    const iconSize = size === 'sm' ? 10 : 12;

    return (
        <button
            id={id}
            type="button"
            role="checkbox"
            aria-checked={indeterminate ? 'mixed' : checked}
            onClick={(e) => { onClick?.(e); onChange(!checked); }}
            className={cn(
                dim,
                "shrink-0 rounded border inline-flex items-center justify-center transition-all duration-150 cursor-pointer",
                checked || indeterminate
                    ? "bg-th-accent border-th-accent text-white"
                    : "bg-th-raised border-th-overlay hover:border-th-accent/50",
                className,
            )}
        >
            {indeterminate ? <Minus size={iconSize} strokeWidth={3} /> : checked ? <Check size={iconSize} strokeWidth={3} /> : null}
        </button>
    );
};
