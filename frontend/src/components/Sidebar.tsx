import React, { useState, useCallback } from "react";
import { LayoutGrid, Activity, Settings, Clock, Zap, ChevronsLeft, ChevronsRight } from "lucide-react";
import { cn } from "../utils";

type SidebarProps = {
  activeTab: string;
  setActiveTab: (tab: string) => void;
  collapsed: boolean;
  onToggleCollapse: () => void;
};

export function Sidebar({ activeTab, setActiveTab, collapsed, onToggleCollapse }: SidebarProps) {
  const [hovered, setHovered] = useState(false);

  const menuItems = [
    { id: "all", label: "Dashboard", icon: LayoutGrid },
    { id: "scheduler", label: "Scheduler", icon: Clock },
    { id: "speedtest", label: "Speed Test", icon: Zap },
    { id: "analytics", label: "Analytics", icon: Activity },
  ];

  const handleDoubleClick = useCallback((e: React.MouseEvent) => {
    // Only toggle if double-clicking on the sidebar itself, not a nav button
    if ((e.target as HTMLElement).closest('button')) return;
    onToggleCollapse();
  }, [onToggleCollapse]);

  const renderNavButton = (item: { id: string; label: string; icon: React.ElementType }) => (
    <button
      key={item.id}
      onClick={() => setActiveTab(item.id)}
      title={collapsed ? item.label : undefined}
      className={cn(
        "w-full flex items-center rounded-lg text-sm font-medium transition-all duration-200 group overflow-hidden",
        collapsed ? "justify-center px-0 py-2.5 gap-0" : "px-3 py-2.5 gap-3",
        activeTab === item.id
          ? "bg-th-raised text-th-accent-t shadow-sm"
          : "text-th-text-s hover:bg-th-raised/50 hover:text-th-text"
      )}
    >
      <item.icon size={18} className={cn("shrink-0 transition-colors", activeTab === item.id ? "text-th-accent-t" : "text-th-text-m group-hover:text-th-text-s")} />
      <span className={cn(
        "whitespace-nowrap transition-all duration-300",
        collapsed ? "w-0 opacity-0" : "w-auto opacity-100"
      )}>{item.label}</span>
    </button>
  );

  return (
    <div
      className="fixed top-0 bottom-0 left-0 z-50"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <aside
        className={cn(
          "bg-th-surface border-r border-th-border flex flex-col h-full transition-all duration-300 ease-in-out",
          collapsed ? "w-16" : "w-64"
        )}
        onDoubleClick={handleDoubleClick}
      >
        {/* Brand */}
        <div className="h-16 flex items-center px-4 border-b border-th-border gap-3 overflow-hidden">
          <div className="w-8 h-8 bg-th-accent rounded-lg flex items-center justify-center shrink-0 shadow-lg shadow-th-accent/20">
            <span className="font-bold text-white text-lg">T</span>
          </div>
          <h1 className={cn(
            "text-xl font-bold tracking-tight text-th-text whitespace-nowrap transition-all duration-300",
            collapsed ? "w-0 opacity-0" : "w-auto opacity-100"
          )}>TDM</h1>
        </div>

        {/* Nav */}
        <nav className="flex-1 px-2 py-6 space-y-1">
          {menuItems.map(renderNavButton)}
        </nav>

        {/* Settings — same spacing as nav */}
        <div className="px-2 pb-2 border-t border-th-border pt-2">
          {renderNavButton({ id: "settings", label: "Settings", icon: Settings })}
        </div>
      </aside>

      {/* Floating Collapse Toggle — overlaps content, visible on hover */}
      <button
        onClick={onToggleCollapse}
        className={cn(
          "absolute top-1/2 -translate-y-1/2 w-5 h-10 flex items-center justify-center z-[60]",
          "bg-th-surface border border-l-0 border-th-border rounded-r-md",
          "text-th-text-m hover:text-th-text hover:bg-th-raised",
          "shadow-sm hover:shadow-md",
          "transition-all duration-300 ease-in-out",
          collapsed ? "left-16" : "left-64",
          hovered ? "opacity-100 translate-x-0" : "opacity-0 -translate-x-1 pointer-events-none"
        )}
        title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
      >
        {collapsed ? <ChevronsRight size={14} /> : <ChevronsLeft size={14} />}
      </button>
    </div>
  );
}
