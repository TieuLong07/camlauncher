import React, { useState, useEffect } from 'react';
import { Titlebar } from './components/Titlebar';
import { Sidebar, NavTab } from './components/Sidebar';
import { PlayView } from './components/PlayView';
import { InstancesView } from './components/InstancesView';
import {
  RECOMMENDED_SERVERS,
  RecommendedServerItem,
} from './data/servers';
import { ModStoreView } from './components/ModStoreView';
import { AccountsView } from './components/AccountsView';
import { ConsoleView } from './components/ConsoleView';
import { SettingsView } from './components/SettingsView';
import { LauncherUpdateModal } from './components/LauncherUpdateModal';
import { InstanceSettingsModal } from './components/InstanceSettingsModal';
import {
  GetSystemStats,
  GetAccounts,
  GetActiveAccount,
  GetInstances,
  LaunchGame,
  KillGame,
  InstallFPSBoostPack,
  OpenInstanceFolder,
  OpenFolder,
  GetCommunityServers,
  InstallCommunityServer,
  SyncCommunityServer,
  CheckLauncherUpdate,
  CreateInstance,
  EnsureOfficialChillTown,
  SyncOfficialChillTown,
  GetTunnelStatus,
  StartTunnel,
  StopTunnel,
  ToggleTunnel,
  CheckEnvironmentSetup,
  AutoSetupEnvironment,
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { models, tunnel } from '../wailsjs/go/models';
import { LogEvent, ProgressEvent } from './types';
import {
  FolderOpen,
  Settings,
  Sparkles,
  Play,
  RotateCw,
} from 'lucide-react';

export const App: React.FC = () => {
  const [currentTab, setCurrentTab] = useState<NavTab>('servers');
  const [tunnelStatus, setTunnelStatus] = useState<tunnel.TunnelStatus | null>(null);
  const [stats, setStats] = useState<models.SystemStats | null>(null);
  const [accounts, setAccounts] = useState<models.Account[]>([]);
  const [activeAccount, setActiveAccount] = useState<models.Account | null>(null);
  const [instances, setInstances] = useState<models.Instance[]>([]);
  const [activeInstance, setActiveInstance] = useState<models.Instance | null>(null);
  const [communityServers, setCommunityServers] = useState<models.CommunityServerEntry[]>([]);
  const [launcherUpdate, setLauncherUpdate] = useState<models.LauncherUpdateInfo | null>(null);
  const [showLauncherUpdateModal, setShowLauncherUpdateModal] = useState(false);
  const [showInstanceSettingsModal, setShowInstanceSettingsModal] = useState(false);
  const [envSetup, setEnvSetup] = useState<models.EnvironmentSetupStatus | null>(null);
  const [isAutoSettingUp, setIsAutoSettingUp] = useState(false);
  const [isRunning, setIsRunning] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [logs, setLogs] = useState<LogEvent[]>([]);
  const [progress, setProgress] = useState<ProgressEvent | null>(null);
  const [toastMsg, setToastMsg] = useState<string | null>(null);

  // Selected Recommended Server
  const [selectedServer, setSelectedServer] = useState<RecommendedServerItem>(
    RECOMMENDED_SERVERS[0]
  );
  const [activeCategory, setActiveCategory] = useState<'recommended' | 'local'>('recommended');
  const [ambientWallpaper, setAmbientWallpaper] = useState<string>(
    RECOMMENDED_SERVERS[0].bannerUrl
  );

  // Dock details matching concept UI
  const [dockData, setDockData] = useState({
    iconKey: 'flask',
    title: RECOMMENDED_SERVERS[0].title,
    isOfficial: true,
    badgeText: 'CHÍNH THỨC',
    badgeClass: 'text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-300 font-mono border border-emerald-500/20',
    version: RECOMMENDED_SERVERS[0].version,
    subtitle: `${RECOMMENDED_SERVERS[0].loader.toUpperCase()} • 8 GB RAM`,
    btnText: 'VÀO MÁY CHỦ NGAY',
  });

  const showToast = (msg: string) => {
    setToastMsg(msg);
    setTimeout(() => setToastMsg(null), 4000);
  };

  const refreshData = async () => {
    try {
      const [sysStats, accs, actAcc, insts, servers] = await Promise.all([
        GetSystemStats(),
        GetAccounts(),
        GetActiveAccount(),
        GetInstances(),
        GetCommunityServers(),
      ]);

      setStats(sysStats);
      setAccounts(accs || []);
      setActiveAccount(actAcc);
      setInstances(insts || []);
      setCommunityServers(servers || []);

      // Home always binds only the official Chill Town profile, never a personal instance.
      if (insts && !activeInstance) {
        const chillTown = insts.find(
          (i) => i.serverId === 'chilltown' || i.name.toLowerCase().includes('chill town') || i.name.toLowerCase().includes('cooking tq')
        );
        if (chillTown) setActiveInstance(chillTown);
      }
    } catch (err) {
      console.error('Failed to load launcher state:', err);
    }
  };

  useEffect(() => {
    refreshData();

    CheckLauncherUpdate()
      .then((res: models.LauncherUpdateInfo) => {
        if (res?.hasUpdate) {
          setLauncherUpdate(res);
        }
      })
      .catch((err: any) => console.error(err));

    const cancelProgress = EventsOn('game:progress', (evt: any) => {
      setProgress(evt);
    });

    const cancelLog = EventsOn('game:log', (logEvt: LogEvent) => {
      setLogs((prev) => {
        const next = [...prev, logEvt];
        if (next.length > 600) return next.slice(next.length - 500);
        return next;
      });
    });

    const cancelStatus = EventsOn('game:status', (statusEvt: any) => {
      setIsRunning(Boolean(statusEvt.running));
      if (!statusEvt.running && statusEvt.message) {
        showToast(statusEvt.message);
      }
    });

    GetTunnelStatus().then(setTunnelStatus).catch(console.error);

    CheckEnvironmentSetup()
      .then(setEnvSetup)
      .catch((e) => console.warn('CheckEnvironmentSetup error:', e));

    const cancelTunnel = EventsOn('tunnel:status', (tStatus: tunnel.TunnelStatus) => {
      setTunnelStatus(tStatus);
    });

    return () => {
      cancelProgress();
      cancelLog();
      cancelStatus();
      cancelTunnel();
    };
  }, []);

  const handleAutoSetupJava = async () => {
    setIsAutoSettingUp(true);
    showToast('Đang tự động tải và cài đặt Java 17 (Adoptium JRE) tối ưu cho Chill Town...');
    try {
      const res = await AutoSetupEnvironment();
      setEnvSetup(res);
      await refreshData();
      showToast('Đã hoàn tất cài đặt Java 17! Môi trường Chill Town đã sẵn sàng.');
    } catch (err: any) {
      showToast(`Lỗi thiết lập Java: ${err}`);
    } finally {
      setIsAutoSettingUp(false);
    }
  };

  const handleToggleTunnel = async () => {
    try {
      const next = await ToggleTunnel();
      setTunnelStatus(next);
      if (next.running) {
        showToast(`Đã bật Cam Tunnel P2P (127.0.0.1:${next.localPort})`);
      } else {
        showToast('Đã tắt Cam Tunnel P2P');
      }
    } catch (err: any) {
      showToast(`Lỗi Cam Tunnel: ${err}`);
    }
  };

  // When selecting a server from the Homescreen
  const handleSelectServer = (server: RecommendedServerItem) => {
    if (isRunning || isSyncing) return;
    setSelectedServer(server);
    setActiveCategory('recommended');
    setAmbientWallpaper(server.bannerUrl);

    // The only recommended server is Chill Town; never bind a personal profile here.
    const officialInst = instances.find(
      (i) => i.serverId === 'chilltown' || i.name.toLowerCase().includes('chill town') || i.name.toLowerCase().includes('cooking tq')
    );
    if (officialInst) {
      setActiveInstance(officialInst);
    }

    setDockData({
      iconKey: server.iconKey,
      title: server.title,
      isOfficial: true,
      badgeText: 'CHÍNH THỨC',
      badgeClass: 'text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-300 font-mono border border-emerald-500/20',
      version: server.version,
      subtitle: `${server.loader.toUpperCase()} • ${server.pingMs}ms P2P`,
      btnText: server.btnText,
    });
  };

  // When selecting an instance from "Bản cài đặt cá nhân"
  const handleSelectLocalInstance = (inst: models.Instance) => {
    if (isRunning || isSyncing) return;
    setActiveInstance(inst);
    setActiveCategory('local');

    const loaderName = inst.loader ? inst.loader.toUpperCase() : 'VANILLA';
    const ramGb = Math.round(inst.maxMemoryMb / 1024);
    const iconKey = inst.loader === 'forge' ? 'flask' : inst.loader === 'fabric' ? 'cube' : 'house';

    setDockData({
      iconKey,
      title: inst.name,
      isOfficial: false,
      badgeText: 'CỤC BỘ',
      badgeClass: 'text-[10px] px-2 py-0.5 rounded-full bg-slate-500/20 text-slate-300 font-mono border border-slate-500/30',
      version: `${loaderName} ${inst.gameVersion}`,
      subtitle: `${loaderName} • ${ramGb} GB RAM`,
      btnText: 'CHƠI NGAY',
    });
    showToast(`Đang chọn bản cài: ${inst.name}`);
  };

  const handleLaunchGame = async (targetInst?: models.Instance) => {
    const inst = targetInst || activeInstance;
    if (!inst) {
      showToast('Vui lòng chọn hoặc tạo một phiên bản để chơi!');
      return;
    }
    if (!activeAccount) {
      showToast('Vui lòng thêm hoặc chọn tài khoản người chơi!');
      setCurrentTab('accounts');
      return;
    }

    try {
      showToast(`Đang chuẩn bị khởi chạy ${inst.name}...`);
      await LaunchGame(inst.id);
    } catch (err: any) {
      showToast(`Lỗi: ${err}`);
    }
  };

  const handleKillGame = async () => {
    try {
      await KillGame();
      showToast('Đã dừng tiến trình Minecraft.');
    } catch (err: any) {
      showToast(`Không thể dừng game: ${err}`);
    }
  };

  const handleOpenFolder = async () => {
    try {
      if (activeInstance && activeInstance.id) {
        await OpenInstanceFolder(activeInstance.id);
        showToast(`Đã mở thư mục: ${activeInstance.name}`);
      } else {
        await OpenFolder('');
        showToast('Đã mở thư mục gốc CamLauncher');
      }
    } catch (err: any) {
      showToast(`Không thể mở thư mục: ${err}`);
    }
  };

  const handleDockPlayAction = async () => {
    if (isRunning) {
      await handleKillGame();
      return;
    }

    if (activeCategory === 'recommended') {
      // Auto-start Cam P2P tunnel if not already running
      try {
        const cur = await GetTunnelStatus();
        if (!cur.running) {
          showToast('Đang tự động bật Cam Tunnel P2P kết nối máy chủ...');
          const started = await StartTunnel();
          setTunnelStatus(started);
          showToast(`Cam Tunnel sẵn sàng (127.0.0.1:${started.localPort})`);
        }
      } catch (tErr) {
        console.warn('Auto start tunnel error:', tErr);
      }

      // Recommended Play always installs/migrates and launches Chill Town.
      // Never fall back to the first personal instance (for example TiCa Boosted).
      try {
        setIsSyncing(true);
        showToast('Đang chuẩn bị Chill Town và kiểm tra modpack chính thức...');
        const official = await EnsureOfficialChillTown();
        await refreshData();
        setActiveInstance(official);
        await handleLaunchGame(official);
      } catch (syncErr: any) {
        showToast(`Không thể chuẩn bị Chill Town: ${syncErr}`);
      } finally {
        setIsSyncing(false);
      }
    } else {
      if (activeInstance) {
        await handleLaunchGame(activeInstance);
      } else {
        showToast('Vui lòng chọn một bản cài đặt để chơi!');
      }
    }
  };

  return (
    <div className="bg-[#0a0d14] text-slate-100 font-sans antialiased select-none h-screen w-screen overflow-hidden flex flex-col justify-between">
      {/* Background Ambient Blur Backdrop */}
      <div className="fixed inset-0 pointer-events-none z-0">
        <div
          id="ambientBg"
          className="w-full h-full bg-cover bg-center opacity-15 filter blur-xl scale-105 transition-all duration-700"
          style={{ backgroundImage: `url('${ambientWallpaper}')` }}
        />
        <div className="absolute inset-0 bg-gradient-to-t from-[#0a0d14] via-[#0a0d14]/92 to-[#0a0d14]/75" />
        <div className="absolute inset-0 bg-gradient-to-r from-[#0a0d14] via-transparent to-[#0a0d14]/85" />
      </div>

      {/* App Main Container */}
      <div className="relative z-10 flex flex-col h-full">
        {/* Top Titlebar */}
        <Titlebar
          stats={stats}
          activeAccount={activeAccount}
          launcherUpdate={launcherUpdate}
          onOpenAccounts={() => setCurrentTab('accounts')}
          onOpenLauncherUpdate={() => setShowLauncherUpdateModal(true)}
          isRunning={isRunning}
        />

        {/* Auto First-Run Setup Banner for New Machine */}
        {envSetup && !envSetup.chillTownReady && (
          <div className="bg-gradient-to-r from-amber-500/20 via-slate-900 to-emerald-500/20 border-b border-amber-500/30 px-6 py-2.5 flex items-center justify-between text-xs z-30 shrink-0">
            <div className="flex items-center gap-2.5 text-amber-200">
              <span className="w-2.5 h-2.5 rounded-full bg-amber-400 animate-pulse" />
              <span className="font-extrabold uppercase tracking-wide text-amber-300">
                Thiết Lập Máy Mới:
              </span>
              <span className="text-slate-200">
                Chưa phát hiện Java 17 (Bắt buộc cho máy chủ Chill Town & Forge 1.20.1).
              </span>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={handleAutoSetupJava}
                disabled={isAutoSettingUp}
                className="status-pill px-4 py-1.5 rounded-xl bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 font-black text-xs transition shadow-md shadow-emerald-500/20 cursor-pointer flex items-center gap-1.5"
              >
                {isAutoSettingUp ? (
                  <>
                    <RotateCw className="w-3.5 h-3.5 animate-spin" />
                    <span>Đang Tự Động Thiết Lập...</span>
                  </>
                ) : (
                  <>
                    <Sparkles className="w-3.5 h-3.5" />
                    <span>Cài Đặt Java 17 Tự Động (1-Click)</span>
                  </>
                )}
              </button>
            </div>
          </div>
        )}

        <div className="flex-1 flex overflow-hidden">
          {/* Sidebar Navigation */}
          <Sidebar
            currentTab={currentTab}
            onSelectTab={setCurrentTab}
            activeAccount={activeAccount}
            activeInstance={activeInstance}
            instanceCount={instances.length}
            isRunning={isRunning}
            onKillGame={handleKillGame}
            onOpenFolder={handleOpenFolder}
          />

          {/* Main Content Pane */}
          <div className="flex-1 flex flex-col overflow-hidden relative">
            {/* Tab 1: Máy chủ đề xuất (Full Homescreen Spotlight) */}
            {currentTab === 'servers' && (
              <PlayView
                selectedServer={selectedServer}
                onSelectServer={handleSelectServer}
                tunnelStatus={tunnelStatus}
                onToggleTunnel={handleToggleTunnel}
                showToast={showToast}
              />
            )}

            {/* Tab 2: Bản cài đặt cá nhân (Grid & Local Management) */}
            {currentTab === 'instances' && (
              <InstancesView
                instances={instances}
                activeInstance={activeInstance}
                stats={stats}
                onSelectInstance={handleSelectLocalInstance}
                onOpenSettings={(inst) => {
                  setActiveInstance(inst);
                  setShowInstanceSettingsModal(true);
                }}
                onRefresh={refreshData}
                showToast={showToast}
              />
            )}

            {/* Tab 3: Kho Mods */}
            {currentTab === 'mods' && (
              <ModStoreView
                activeInstance={activeInstance}
                instances={instances}
                onSelectInstance={(inst) => {
                  setActiveInstance(inst);
                  handleSelectLocalInstance(inst);
                }}
              />
            )}

            {/* Tab 4: Quản lý tài khoản */}
            {currentTab === 'accounts' && (
              <AccountsView
                accounts={accounts}
                activeAccount={activeAccount}
                onRefresh={refreshData}
              />
            )}

            {/* Tab 5: Nhật ký Console */}
            {currentTab === 'console' && (
              <ConsoleView logs={logs} onClearLogs={() => setLogs([])} isRunning={isRunning} />
            )}

            {/* Tab 6: Cài đặt hệ thống */}
            {currentTab === 'settings' && (
              <SettingsView
                stats={stats}
                launcherUpdate={launcherUpdate}
                onCheckLauncherUpdate={() => CheckLauncherUpdate().then((r: any) => setLauncherUpdate(r))}
                onOpenLauncherUpdate={() => setShowLauncherUpdateModal(true)}
              />
            )}
          </div>
        </div>

        {/* BOTTOM ACTION DOCK matching Mockup in Images 1 & 2 */}
        <footer
          className="glass-panel flex items-center justify-between border-t border-white/10 relative z-20 shrink-0"
          style={{ height: '82px', padding: '0 24px' }}
        >
          {/* Left: Selected Instance Details */}
          <div className="flex items-center gap-3.5">
            <div
              className="w-12 h-12 rounded-2xl bg-white/[0.04] border border-white/10 flex items-center justify-center shadow-inner relative shrink-0"
            >
              <span className="text-xl">
                {dockData.iconKey === 'flask' ? (
                  <svg className="w-6 h-6 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                  </svg>
                ) : dockData.iconKey === 'swords' ? (
                  <svg className="w-6 h-6 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                ) : (
                  <svg className="w-6 h-6 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                  </svg>
                )}
              </span>

              <span
                className={`absolute -top-1 -right-1 w-3 h-3 ${
                  dockData.isOfficial ? 'bg-emerald-500' : 'bg-slate-400'
                } border-2 border-[#0a0d14] rounded-full`}
              />
            </div>

            <div className="flex flex-col text-left gap-0.5">
              <div className="flex items-center gap-2">
                <span className="text-base font-extrabold text-white tracking-wide truncate max-w-xs">
                  {dockData.title}
                </span>
                <span
                  className={`status-pill ${dockData.badgeClass}`}
                  style={{
                    height: '22px',
                    padding: '0 10px',
                    fontSize: '10px',
                    fontWeight: 700,
                    letterSpacing: '0.04em',
                  }}
                >
                  {dockData.badgeText}
                </span>
              </div>
              <div className="text-xs text-slate-400 flex items-center gap-1.5 mt-0.5">
                <span>{dockData.version}</span>
                <span>•</span>
                <span className={dockData.isOfficial ? 'text-emerald-400 font-semibold' : 'text-slate-400'}>
                  {dockData.subtitle}
                </span>
              </div>
            </div>
          </div>

          {/* Right: Actions and Primary CTA */}
          <div className="flex items-center gap-3">
            {/* Open Folder */}
            <button
              onClick={handleOpenFolder}
              title={`Mở thư mục instance: ${dockData.title}`}
              className="status-pill group rounded-2xl bg-white/[0.04] hover:bg-emerald-500/10 border border-white/10 hover:border-emerald-500/35 text-slate-300 hover:text-emerald-300 transition-all duration-200 cursor-pointer shadow-sm active:scale-95 flex items-center gap-2"
              style={{
                height: '50px',
                padding: '0 18px',
                fontSize: '13px',
                fontWeight: 600,
              }}
            >
              <FolderOpen className="w-4 h-4 text-emerald-400 group-hover:scale-110 transition-transform" />
              <span className="hidden sm:inline font-semibold">Thư Mục</span>
            </button>

            {/* Instance Settings */}
            <button
              onClick={() => {
                if (activeInstance) {
                  setShowInstanceSettingsModal(true);
                } else {
                  showToast('Vui lòng chọn hoặc tạo bản cài trước khi chỉnh cài đặt!');
                }
              }}
              title={`Cài đặt instance: ${dockData.title}`}
              className="status-pill group rounded-2xl bg-white/[0.04] hover:bg-cyan-500/10 border border-white/10 hover:border-cyan-500/35 text-slate-300 hover:text-cyan-300 transition-all duration-200 cursor-pointer shadow-sm active:scale-95 flex items-center gap-2"
              style={{
                height: '50px',
                padding: '0 18px',
                fontSize: '13px',
                fontWeight: 600,
              }}
            >
              <Settings className="w-4 h-4 text-cyan-400 group-hover:scale-110 group-hover:rotate-45 transition-transform duration-300" />
              <span className="hidden sm:inline font-semibold">Cài Đặt</span>
            </button>

            {/* Primary Action Button (Play/Stop) */}
            <button
              id="playButton"
              onClick={handleDockPlayAction}
              disabled={isSyncing}
              className={`relative w-64 md:w-72 h-13 py-3.5 px-6 rounded-2xl transition-all duration-300 shadow-lg flex items-center justify-center overflow-hidden font-bold text-base tracking-wide cursor-pointer ${
                isRunning
                  ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30'
                  : isSyncing
                  ? 'bg-slate-800 text-emerald-300 border border-emerald-500/40'
                  : 'bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 active:scale-[0.98] text-slate-950 shadow-emerald-500/25'
              }`}
            >
              <div
                id="progressBar"
                className="absolute inset-y-0 left-0 bg-emerald-300/30 transition-all duration-150 ease-out"
                style={{
                  width: `${progress?.percentage ? Math.min(100, Math.max(0, progress.percentage)) : 0}%`,
                }}
              />
              <div id="buttonContent" className="relative z-10 flex items-center gap-2">
                {isRunning ? (
                  <>
                    <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 animate-ping mr-1" />
                    <span className="text-xs font-bold uppercase tracking-wider">
                      ĐANG CHƠI (BẤM ĐỂ DỪNG)
                    </span>
                  </>
                ) : isSyncing ? (
                  <>
                    <RotateCw className="w-4 h-4 animate-spin text-emerald-400" />
                    <span className="font-mono text-xs font-semibold">
                      ĐỒNG BỘ MOD ({Math.round(progress?.percentage || 0)}%)
                    </span>
                  </>
                ) : (
                  <>
                    <Play className="w-5 h-5 fill-current" />
                    <span id="buttonText" className="tracking-wider font-extrabold uppercase text-sm">
                      {dockData.btnText}
                    </span>
                  </>
                )}
              </div>
            </button>
          </div>
        </footer>
      </div>

      {/* Launcher Auto-Update Modal */}
      {showLauncherUpdateModal && launcherUpdate && (
        <LauncherUpdateModal
          updateInfo={launcherUpdate}
          onClose={() => setShowLauncherUpdateModal(false)}
          progress={progress}
        />
      )}

      {/* Instance Settings Modal */}
      <InstanceSettingsModal
        instance={activeInstance}
        stats={stats}
        isOpen={showInstanceSettingsModal}
        onClose={() => setShowInstanceSettingsModal(false)}
        onUpdated={(updatedInst) => {
          refreshData();
          setActiveInstance(updatedInst);
          setDockData((prev) => ({
            ...prev,
            title: updatedInst.name,
            version: `${updatedInst.loader ? updatedInst.loader.toUpperCase() : 'VANILLA'} ${updatedInst.gameVersion}`,
            subtitle: `${updatedInst.loader ? updatedInst.loader.toUpperCase() : 'Mojang Official'} • ${Math.round(updatedInst.maxMemoryMb / 1024)} GB RAM`,
          }));
        }}
        showToast={showToast}
      />

      {/* Toast Notification */}
      {toastMsg && (
        <div className="fixed top-18 right-6 z-50 px-4 py-2.5 rounded-xl bg-slate-900/90 text-emerald-300 border border-emerald-500/30 backdrop-blur-md shadow-2xl text-xs font-medium animate-fadeIn flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-emerald-400" />
          <span>{toastMsg}</span>
        </div>
      )}
    </div>
  );
};

export default App;
