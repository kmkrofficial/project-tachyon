import { LayoutDashboard, Download, CheckCircle, List } from "lucide-react";
import clsx from "clsx";

type SidebarProps = {
  activeTab: string;
  setActiveTab: (tab: string) => void;
};

export function Sidebar({ activeTab, setActiveTab }: SidebarProps) {
  const menuItems = [
    { id: "all", label: "All Downloads", icon: LayoutDashboard },
    { id: "downloading", label: "Downloading", icon: Download },
    { id: "completed", label: "Completed", icon: CheckCircle },
    { id: "queued", label: "Queued", icon: List },
  ];

  return (
    <div className="w-64 h-screen bg-gray-900 text-white flex flex-col border-r border-gray-800">
      <div className="p-6">
        <h1 className="text-2xl font-bold tracking-tight text-blue-500">Tachyon</h1>
      </div>
      <nav className="flex-1 px-4 space-y-2">
        {menuItems.map((item) => (
          <button
            key={item.id}
            onClick={() => setActiveTab(item.id)}
            className={clsx(
              "w-full flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors",
              activeTab === item.id
                ? "bg-blue-600 text-white shadow-lg shadow-blue-900/50"
                : "text-gray-400 hover:bg-gray-800 hover:text-white"
            )}
          >
            <item.icon size={20} />
            {item.label}
          </button>
        ))}
      </nav>
      <div className="p-4 border-t border-gray-800">
        <div className="text-xs text-gray-500 text-center">
          v0.1.0-alpha
        </div>
      </div>
    </div>
  );
}
