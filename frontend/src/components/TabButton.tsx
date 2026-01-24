import React from 'react';
import { LucideIcon } from 'lucide-react';

interface TabButtonProps {
    id: string;
    label: string;
    icon: LucideIcon;
    active: boolean;
    onClick: () => void;
}

export const TabButton: React.FC<TabButtonProps> = ({ label, icon: Icon, active, onClick }) => (
    <button
        onClick={onClick}
        className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${active ? "bg-gray-800 text-white" : "text-gray-400 hover:text-gray-200"}`}
    >
        <Icon size={16} />
        {label}
    </button>
);
