import React, { useState } from 'react';
import { models } from '../../wailsjs/go/models';
import {
  Compass,
  Layers,
  Package,
  Users,
  Settings,
  ChevronsLeft,
  ChevronsUpDown,
} from 'lucide-react';

export type NavTab = 'servers' | 'instances' | 'mods' | 'accounts' | 'console' | 'settings';

interface SidebarProps {
  currentTab: NavTab;
  onSelectTab: (tab: NavTab) => void;
  activeAccount: models.Account | null;
  activeInstance: models.Instance | null;
  instanceCount?: number;
  isRunning?: boolean;
  onKillGame?: () => void;
  onOpenFolder?: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  onSelectTab,
  activeAccount,
  instanceCount = 0,
}) => {
  const [collapsed, setCollapsed] = useState(false);

  const navItems = [
    {
      id: 'servers' as NavTab,
      label: 'Máy chủ đề xuất',
      badge: null,
      count: null as number | null,
      icon: <Compass className="w-4 h-4 shrink-0" />,
    },
    {
      id: 'instances' as NavTab,
      label: 'Bản cài đặt cá nhân',
      badge: 'BETA',
      count: instanceCount > 0 ? instanceCount : null,
      icon: <Layers className="w-4 h-4 shrink-0" />,
    },
    {
      id: 'mods' as NavTab,
      label: 'Kho Mods',
      badge: 'BETA',
      count: null as number | null,
      icon: <Package className="w-4 h-4 shrink-0" />,
    },
    {
      id: 'accounts' as NavTab,
      label: 'Tài khoản',
      badge: null,
      count: null as number | null,
      icon: <Users className="w-4 h-4 shrink-0" />,
    },
    {
      id: 'settings' as NavTab,
      label: 'Cài đặt',
      badge: null,
      count: null as number | null,
      icon: <Settings className="w-4 h-4 shrink-0" />,
    },
  ];

  const accountName = activeAccount?.username || 'CamPlayer';
  const accountType = activeAccount?.type === 'microsoft' ? 'Microsoft' : 'Offline';
  const avatarUrl = `https://minotar.net/helm/${accountName}/36.png`;

  return (
    <aside
      id="mainSidebar"
      style={{ padding: '18px 14px', boxSizing: 'border-box' }}
      className={`${
        collapsed ? 'w-20' : 'w-64'
      } flex flex-col justify-between border-r border-white/5 bg-[#0a0d14]/70 backdrop-blur-xl shrink-0 transition-all duration-300 ease-in-out select-none z-20`}
    >
      <nav className="flex flex-col" style={{ gap: '6px' }}>
        {/* Sidebar Header with Collapse Button */}
        <div
          className="flex items-center justify-between px-2 mb-2 text-[11px] font-bold tracking-widest text-slate-500 uppercase select-none"
          style={{ height: '32px' }}
        >
          {!collapsed && <span className="transition-opacity duration-200">Danh Mục</span>}
          <button
            onClick={() => setCollapsed(!collapsed)}
            title={collapsed ? 'Mở rộng menu' : 'Thu gọn menu'}
            className="w-7 h-7 rounded-lg hover:bg-white/10 flex items-center justify-center text-slate-400 hover:text-white transition-colors ml-auto cursor-pointer"
          >
            <ChevronsLeft className={`w-4 h-4 transition-transform duration-300 ${collapsed ? 'rotate-180' : ''}`} />
          </button>
        </div>

        {/* Navigation Buttons */}
        {navItems.map((item) => {
          const isActive = currentTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => onSelectTab(item.id)}
              title={item.label}
              style={{
                height: '46px',
                padding: collapsed ? '0' : '0 14px',
                gap: '12px',
                borderRadius: '14px',
              }}
              className={`w-full flex items-center ${
                collapsed ? 'justify-center' : 'justify-between'
              } text-[13.5px] font-semibold transition-all cursor-pointer ${
                isActive
                  ? 'bg-emerald-500/15 text-emerald-300 border border-emerald-500/30 shadow-md shadow-emerald-500/10'
                  : 'text-slate-400 hover:text-slate-100 hover:bg-white/[0.04] border border-transparent'
              }`}
            >
              <div className="flex items-center gap-3 min-w-0">
                <span className={`shrink-0 ${isActive ? 'text-emerald-400' : 'text-slate-400'}`}>
                  {item.icon}
                </span>
                {!collapsed && (
                  <span className="whitespace-nowrap overflow-hidden text-ellipsis text-left font-medium">
                    {item.label}
                  </span>
                )}
              </div>

              {!collapsed && (
                <div className="flex items-center gap-1.5 shrink-0">
                  {item.badge && (
                    <span className="status-pill font-mono font-extrabold text-[9px] px-1.5 py-0.5 rounded-md bg-amber-500/15 text-amber-300 border border-amber-500/30">
                      {item.badge}
                    </span>
                  )}
                  {item.count !== null && item.count !== undefined && (
                    <span
                      className={`status-pill font-mono font-bold text-[10.5px] px-2 py-0.5 rounded-full ${
                        isActive
                          ? 'bg-emerald-500/25 text-emerald-300 border border-emerald-500/35'
                          : 'bg-white/10 text-slate-400 border border-white/10'
                      }`}
                    >
                      {item.count}
                    </span>
                  )}
                </div>
              )}
            </button>
          );
        })}
      </nav>

      {/* Bottom Sidebar: User Profile Card */}
      <div style={{ paddingTop: '14px', borderTop: '1px solid rgba(255, 255, 255, 0.08)' }}>
        <button
          onClick={() => onSelectTab('accounts')}
          title={`Tài khoản: ${accountName} (${accountType})`}
          style={{
            height: '56px',
            padding: collapsed ? '0' : '0 12px',
            borderRadius: '14px',
          }}
          className={`w-full flex items-center ${
            collapsed ? 'justify-center' : 'justify-between'
          } bg-white/[0.03] hover:bg-white/[0.08] border border-white/10 hover:border-emerald-500/30 transition-all text-left group cursor-pointer`}
        >
          <div className="flex items-center min-w-0" style={{ gap: '12px' }}>
            <div className="relative shrink-0">
              <img
                src={avatarUrl}
                alt="Avatar"
                className="w-9 h-9 rounded-xl shadow-md border border-white/15 bg-black/40 object-cover pixelated"
                onError={(e) => {
                  (e.target as HTMLElement).style.display = 'none';
                }}
              />
              <span className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 bg-emerald-400 border-2 border-[#0a0d14] rounded-full" />
            </div>
            {!collapsed && (
              <div className="flex flex-col min-w-0 overflow-hidden" style={{ gap: '2px' }}>
                <span className="text-xs font-bold text-slate-100 truncate tracking-wide">
                  {accountName}
                </span>
                <div className="flex items-center">
                  <span
                    className={`status-pill font-mono font-bold text-[9.5px] uppercase tracking-wider rounded-md ${
                      accountType === 'Microsoft'
                        ? 'text-blue-300 bg-blue-500/15 border border-blue-500/25'
                        : 'text-emerald-300 bg-emerald-500/15 border border-emerald-500/25'
                    }`}
                    style={{ height: '18px', padding: '0 6px' }}
                  >
                    {accountType}
                  </span>
                </div>
              </div>
            )}
          </div>
          {!collapsed && (
            <ChevronsUpDown className="w-4 h-4 text-slate-400 shrink-0 ml-1 group-hover:text-emerald-300 transition-colors" />
          )}
        </button>
      </div>
    </aside>
  );
};
