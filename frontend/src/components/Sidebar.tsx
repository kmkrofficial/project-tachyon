import React from "react";
import { LayoutGrid, Download, Activity, Settings, Clock, HardDrive, Zap, PanelLeftClose, PanelLeft } from "lucide-react";
import { cn } from "../utils";

type SidebarProps = {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  diskUsage?: { free_gb: number, percent: number };
  collapsed: boolean;
  onToggleCollapse: () => void;
};

export function Sidebar({ activeTab, setActiveTab, diskUsage = { free_gb: 0, percent: 0 }, collapsed, onToggleCollapse }: SidebarProps) {
  const menuItems = [
    { id: "all", label: "Dashboard", icon: LayoutGrid },
    { id: "downloading", label: "Active", icon: Download },
    { id: "analytics", label: "Analytics", icon: Activity },
    { id: "scheduler", label: "Scheduler", icon: Clock },
    { id: "speedtest", label: "Speed Test", icon: Zap },
    { id: "settings", label: "Settings", icon: Settings },
  ];

  return (
    <aside className={cn(
      "bg-th-surface border-r border-th-border flex flex-col fixed top-0 bottom-0 left-0 z-50 transition-all duration-300",
      collapsed ? "w-16" : "w-64"
    )}>
      {/* Brand */}
      <div className="h-16 flex items-center px-4 border-b border-th-border gap-3">
        <div className="w-8 h-8 bg-gradient-to-br from-cyan-400 to-blue-600 rounded-lg flex items-center justify-center shrink-0 shadow-lg shadow-cyan-900/20">
          <span className="font-bold text-white text-lg">T</span>
        </div>
        {!collapsed && <h1 className="text-xl font-bold tracking-tight text-th-text">Tachyon</h1>}
      </div>

      {/* Nav */}
      <nav className="flex-1 px-2 py-6 space-y-1">
        {menuItems.map((item) => (
          <button
            key={item.id}
            onClick={() => setActiveTab(item.id)}
            title={collapsed ? item.label : undefined}
            className={cn(
              "w-full flex items-center gap-3 rounded-lg text-sm font-medium transition-all group",
              collapsed ? "justify-center px-0 py-2.5" : "px-3 py-2.5",
              activeTab === item.id || (activeTab === 'all' && item.id === 'dashboard')
                ? "bg-th-raised text-cyan-500 shadow-sm"
                : "text-th-text-s hover:bg-th-raised/50 hover:text-th-text"
            )}
          >
            <item.icon size={18} className={cn("shrink-0 transition-colors", activeTab === item.id ? "text-cyan-500" : "text-th-text-m group-hover:text-th-text-s")} />
            {!collapsed && item.label}
          </button>
        ))}
      </nav>

      {/* Disk Usage */}
      {!collapsed && (
        <div className="p-6 border-t border-th-border bg-th-surface/50">
          <div className="flex justify-between text-xs text-th-text-s mb-2">
            <span>Disk Usage</span>
            <span className="text-th-text">{diskUsage.free_gb.toFixed(0)}GB Free</span>
          </div>
          <div className="h-1.5 w-full bg-th-raised rounded-full overflow-hidden">
            <div
              className="h-full bg-gradient-to-r from-cyan-500 to-blue-500 rounded-full transition-all duration-500"
              style={{ width: `${diskUsage.percent}%` }}
            ></div>
          </div>
        </div>
      )}

      {/* Collapse Toggle */}
      <div className="p-2 border-t border-th-border">
        <button
          onClick={onToggleCollapse}
          className="w-full flex items-center justify-center p-2 rounded-lg text-th-text-s hover:bg-th-raised hover:text-th-text transition-colors"
          title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          {collapsed ? <PanelLeft size={18} /> : <PanelLeftClose size={18} />}
        </button>
      </div>
    </aside>
  );
}
