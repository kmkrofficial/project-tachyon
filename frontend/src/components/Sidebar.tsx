import React from "react";
import { LayoutGrid, Download, CheckCircle, List, Activity, Settings, Clock, HardDrive, Zap } from "lucide-react";
import { cn } from "../utils";

type SidebarProps = {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  diskUsage?: { free_gb: number, percent: number };
};

export function Sidebar({ activeTab, setActiveTab, diskUsage = { free_gb: 0, percent: 0 } }: SidebarProps) {
  const menuItems = [
    { id: "all", label: "Dashboard", icon: LayoutGrid },
    { id: "downloading", label: "Active", icon: Download },
    // { id: "completed", label: "Finished", icon: CheckCircle },
    { id: "analytics", label: "Analytics", icon: Activity },
    { id: "scheduler", label: "Scheduler", icon: Clock },
    { id: "speedtest", label: "Speed Test", icon: Zap },
    { id: "settings", label: "Settings", icon: Settings },
  ];

  return (
    <aside className="w-64 bg-slate-900 border-r border-slate-800 flex flex-col fixed top-0 bottom-0 left-0 z-50">
      {/* Brand */}
      <div className="h-16 flex items-center px-6 border-b border-slate-800">
        <div className="w-8 h-8 bg-gradient-to-br from-cyan-400 to-blue-600 rounded-lg flex items-center justify-center mr-3 shadow-lg shadow-cyan-900/20">
          <span className="font-bold text-white text-lg">T</span>
        </div>
        <h1 className="text-xl font-bold tracking-tight text-slate-100">Tachyon</h1>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-3 py-6 space-y-1">
        {menuItems.map((item) => (
          <button
            key={item.id}
            onClick={() => setActiveTab(item.id)}
            className={cn(
              "w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all group",
              activeTab === item.id || (activeTab === 'all' && item.id === 'dashboard')
                ? "bg-slate-800 text-cyan-400 shadow-sm"
                : "text-slate-400 hover:bg-slate-800/50 hover:text-slate-200"
            )}
          >
            <item.icon size={18} className={cn("transition-colors", activeTab === item.id ? "text-cyan-400" : "text-slate-500 group-hover:text-slate-300")} />
            {item.label}
          </button>
        ))}
      </nav>

      {/* Disk Usage */}
      <div className="p-6 border-t border-slate-800 bg-slate-900/50">
        <div className="flex justify-between text-xs text-slate-400 mb-2">
          <span>Disk Usage</span>
          <span className="text-slate-200">{diskUsage.free_gb.toFixed(0)}GB Free</span>
        </div>
        <div className="h-1.5 w-full bg-slate-800 rounded-full overflow-hidden">
          <div
            className="h-full bg-gradient-to-r from-cyan-500 to-blue-500 rounded-full transition-all duration-500"
            style={{ width: `${diskUsage.percent}%` }}
          ></div>
        </div>
      </div>
    </aside>
  );
}
