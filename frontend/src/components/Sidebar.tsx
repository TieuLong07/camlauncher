import { useState } from 'react';
import { ChevronLeft, ChevronRight, Compass, Layers, Package, Settings, Terminal, UserRound } from 'lucide-react';
import { models } from '../../wailsjs/go/models';

export type NavTab = 'servers' | 'instances' | 'mods' | 'accounts' | 'console' | 'settings';

const links = [
  { id: 'servers', name: 'Máy chủ', icon: Compass },
  { id: 'instances', name: 'Bản cài cá nhân', icon: Layers },
  { id: 'mods', name: 'Kho Mods', icon: Package },
  { id: 'console', name: 'Nhật ký', icon: Terminal },
  { id: 'settings', name: 'Cài đặt', icon: Settings },
] as const;

interface SidebarProps {
  currentTab: NavTab;
  onSelectTab: (tab: NavTab) => void;
  activeAccount: models.Account | null;
  instanceCount: number;
}

export function Sidebar({ currentTab, onSelectTab, activeAccount, instanceCount }: SidebarProps) {
  const [collapsed, setCollapsed] = useState(false);
  const account = activeAccount?.username.toLowerCase() === 'camplayer' ? null : activeAccount;
  const label = account?.username ?? 'Tài khoản';
  return (
    <aside className={`launcher-sidebar ${collapsed ? 'is-collapsed' : ''}`}>
      <button className="sidebar-toggle cam-button cam-icon" aria-expanded={!collapsed}
        aria-label={collapsed ? 'Mở rộng menu' : 'Thu gọn menu'} title={collapsed ? 'Mở rộng menu' : 'Thu gọn menu'}
        onClick={() => setCollapsed(!collapsed)}>
        {collapsed ? <ChevronRight size={17} /> : <ChevronLeft size={17} />}
      </button>
      <nav aria-label="Điều hướng chính">
        {links.map(({ id, name, icon: Icon }) => (
          <button key={id} className="sidebar-link" aria-current={currentTab === id ? 'page' : undefined}
            title={name} aria-label={name} onClick={() => onSelectTab(id)}>
            <Icon size={18} />
            {!collapsed && <><span>{name}</span>{id === 'instances' && instanceCount > 0 && <small>{instanceCount}</small>}</>}
          </button>
        ))}
      </nav>
      <button className="sidebar-account sidebar-link" aria-current={currentTab === 'accounts' ? 'page' : undefined}
        title={label} aria-label={account ? `Tài khoản ${label}` : 'Tài khoản'}
        onClick={() => onSelectTab('accounts')}>
        <span className="sidebar-avatar">
          <UserRound size={17} />
          {account && <img key={account.id} src={`https://minotar.net/helm/${encodeURIComponent(account.username)}/36.png`} alt=""
            onError={(event) => { event.currentTarget.hidden = true; }} />}
        </span>
        {!collapsed && <span className="sidebar-account-name"><strong>{label}</strong>
          <small>{account ? account.type === 'microsoft' ? 'Microsoft' : 'Offline' : 'Chưa thiết lập'}</small></span>}
      </button>
    </aside>
  );
}
