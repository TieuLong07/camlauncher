import { useEffect, useRef, useState } from 'react';
import {
  AlertCircle, Box, Download, FolderOpen, Loader2, Play, RefreshCw, Settings, Square, Terminal,
} from 'lucide-react';
import {
  AutoSetupEnvironment, CheckEnvironmentSetup, CheckLauncherUpdate, CheckOfficialModpack,
  SyncOfficialModpack, GetAccounts,
  GetActiveAccount, GetAppVersion, GetInstances, GetServerLiveStatus, GetSystemStats, GetTunnelStatus, KillGame, LaunchGame,
  OpenInstanceFolder, StartTunnel, ToggleTunnel,
} from '../wailsjs/go/main/App';
import { models, tunnel } from '../wailsjs/go/models';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { AccountSetupModal } from './components/AccountSetupModal';
import { AccountsView } from './components/AccountsView';
import { ConsoleView } from './components/ConsoleView';
import { InstanceSettingsModal } from './components/InstanceSettingsModal';
import { InstancesView } from './components/InstancesView';
import { LauncherUpdateModal } from './components/LauncherUpdateModal';
import { ModStoreView } from './components/ModStoreView';
import { PlayView } from './components/PlayView';
import { SettingsView } from './components/SettingsView';
import { Sidebar, NavTab } from './components/Sidebar';
import { Titlebar } from './components/Titlebar';
import { RECOMMENDED_SERVERS, RecommendedServerItem } from './data/servers';
import { LogEvent, ProgressEvent } from './types';

type PackCheck = {
  status: 'checking' | 'ready' | 'error';
  result?: models.UpdateCheckResult;
};

type Activity = { kind: 'launch' | 'pack' | 'java'; title: string; label: string };

function getOfficialInstance(instances: models.Instance[], server: RecommendedServerItem) {
  const registered = instances.find((instance) => instance.serverId?.toLowerCase() === server.id);
  if (registered) return registered;
  if (server.isKamaron) {
    return instances.find((instance) => !instance.serverId && /(?:^|[\\/])instances[\\/]kamaron[\\/]*$/i.test(instance.path || '')) ?? null;
  }
  if (server.id !== 'chilltown') return null;
  return instances.find((instance) => !instance.serverId && /chill town|cooking tq/i.test(instance.name)) ?? null;
}

export function App() {
  const [currentTab, setCurrentTab] = useState<NavTab>('servers');
  const [selectedServer, setSelectedServer] = useState(RECOMMENDED_SERVERS[0]);
  const [localId, setLocalId] = useState<string>();
  const [modTargetId, setModTargetId] = useState<string>();
  const [settingsId, setSettingsId] = useState<string>();
  const [instances, setInstances] = useState<models.Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [accounts, setAccounts] = useState<models.Account[]>([]);
  const [activeAccount, setActiveAccount] = useState<models.Account | null>(null);
  const [stats, setStats] = useState<models.SystemStats | null>(null);
  const [appVersion, setAppVersion] = useState('');
  const [launcherUpdate, setLauncherUpdate] = useState<models.LauncherUpdateInfo | null>(null);
  const [checkingLauncher, setCheckingLauncher] = useState(false);
  const [launcherCheckError, setLauncherCheckError] = useState(false);
  const [showLauncherUpdate, setShowLauncherUpdate] = useState(false);
  const [showAccountSetup, setShowAccountSetup] = useState(false);
  const [envSetup, setEnvSetup] = useState<models.EnvironmentSetupStatus | null>(null);
  const [tunnelStatus, setTunnelStatus] = useState<tunnel.TunnelStatus | null>(null);
  const [tunnelBusy, setTunnelBusy] = useState(false);
  const [packChecks, setPackChecks] = useState<Record<string, PackCheck>>({});
  const [activity, setActivity] = useState<Activity | null>(null);
  const [runningTitle, setRunningTitle] = useState('');
  const [stopping, setStopping] = useState(false);
  const [logs, setLogs] = useState<LogEvent[]>([]);
  const [progress, setProgress] = useState<ProgressEvent | null>(null);
  const [actionError, setActionError] = useState<{ title: string; message: string; offlineAvailable?: boolean } | null>(null);
  const [toast, setToast] = useState('');
  const toastTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const currentActivity = useRef<Activity | null>(null);
  const packRequests = useRef<Record<string, number>>({});

  const showToast = (message: string) => {
    clearTimeout(toastTimer.current);
    setToast(message);
    toastTimer.current = setTimeout(() => setToast(''), 4200);
  };

  const refreshData = async () => {
    const [sysStats, allAccounts, account, allInstances] = await Promise.all([
      GetSystemStats(), GetAccounts(), GetActiveAccount(), GetInstances(),
    ]);
    setStats(sysStats);
    setAccounts(allAccounts ?? []);
    setActiveAccount(account ?? null);
    setInstances(allInstances ?? []);
    return { account, allAccounts: allAccounts ?? [] };
  };

  const loadLauncher = async () => {
    setLoading(true);
    setLoadError(false);
    try {
      const { account, allAccounts } = await refreshData();
      if ((!account || account.username.toLowerCase() === 'camplayer') &&
          !allAccounts.some((item) => item.username.toLowerCase() !== 'camplayer')) {
        setShowAccountSetup(true);
      }
    } catch {
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  };

  const refreshDataWithNotice = () => refreshData().catch((error) => {
    showToast(`Không thể nạp lại dữ liệu. ${error}`);
  });

  const refreshEnvironment = () => CheckEnvironmentSetup().then(setEnvSetup).catch(console.warn);

  const checkLauncher = async (atStartup = false) => {
    setCheckingLauncher(true);
    setLauncherCheckError(false);
    try {
      const result = await CheckLauncherUpdate();
      setLauncherUpdate(result);
    } catch (error) {
      setLauncherCheckError(true);
      if (!atStartup) showToast(`Không thể kiểm tra cập nhật launcher. ${error}`);
    } finally {
      setCheckingLauncher(false);
    }
  };

  const checkPack = async (server: RecommendedServerItem) => {
    const request = (packRequests.current[server.id] ?? 0) + 1;
    packRequests.current[server.id] = request;
    setPackChecks((previous) => ({ ...previous, [server.id]: { ...previous[server.id], status: 'checking' } }));
    try {
      const result = await CheckOfficialModpack(server.id);
      if (packRequests.current[server.id] === request) {
        setPackChecks((previous) => ({ ...previous, [server.id]: { status: 'ready', result } }));
      }
    } catch {
      if (packRequests.current[server.id] === request) {
        setPackChecks((previous) => ({ ...previous, [server.id]: { ...previous[server.id], status: 'error' } }));
      }
    }
  };

  const finishActivity = () => {
    currentActivity.current = null;
    setActivity(null);
    setProgress(null);
  };

  useEffect(() => {
    void loadLauncher();
    void checkLauncher(true);
    void refreshEnvironment();
    GetAppVersion().then(setAppVersion).catch(console.warn);
    GetTunnelStatus().then(setTunnelStatus).catch(console.warn);

    const cancelProgress = EventsOn('game:progress', (event: ProgressEvent) => {
      setProgress(event);
    });
    const cancelLog = EventsOn('game:log', (event: LogEvent) => {
      setLogs((previous) => [...previous.slice(-599), event]);
    });
    const cancelStatus = EventsOn('game:status', (event: { running: boolean; error?: string; message?: string }) => {
      const pending = currentActivity.current;
      if (event.running && pending?.kind === 'launch') setRunningTitle(pending.title);
      if (!event.running) {
        setRunningTitle('');
        setStopping(false);
        if (event.error) setActionError({ title: pending?.title ?? 'Minecraft', message: event.error });
        else if (event.message) showToast(event.message);
      }
      if (pending?.kind === 'launch') finishActivity();
    });
    const cancelTunnel = EventsOn('tunnel:status', setTunnelStatus);
    return () => {
      cancelProgress();
      cancelLog();
      cancelStatus();
      cancelTunnel();
      clearTimeout(toastTimer.current);
    };
  }, []);

  const officialInstance = getOfficialInstance(instances, selectedServer);
  const localInstance = instances.find((instance) => instance.id === localId) ?? null;
  const modTarget = instances.find((instance) => instance.id === modTargetId) ?? localInstance ?? officialInstance ?? instances[0] ?? null;
  const settingsInstance = instances.find((instance) => instance.id === settingsId) ?? null;
  const fromServer = currentTab === 'servers';
  const playableInstance = fromServer ? officialInstance : localInstance;
  const showPlay = fromServer || currentTab === 'instances';
  const title = fromServer ? selectedServer.title : localInstance?.name ?? 'Chọn một bản cài';
  const pack = packChecks[selectedServer.id];
  const needsJava = fromServer && envSetup !== null && !envSetup.hasJava17;
  const needsAccount = !activeAccount || activeAccount.username.toLowerCase() === 'camplayer';
  const legacyKamaron = Boolean(selectedServer.isKamaron && officialInstance && !officialInstance.serverId);
  const packAction = !officialInstance ? 'install' : legacyKamaron || pack?.result?.isDrifted ? 'sync' : pack?.result?.hasUpdate ? 'update' : null;
  const action = fromServer && pack?.status === 'error' ? 'check' : needsJava ? 'java' : fromServer && packAction ? packAction : needsAccount ? 'account' : 'play';
  const packStatus = !officialInstance
    ? 'Chưa cài modpack'
    : pack?.status === 'checking' || !pack
      ? 'Đang kiểm tra modpack…'
      : pack.status === 'error'
        ? 'Chưa kiểm tra được modpack'
        : legacyKamaron ? 'Đã tìm thấy bản cài cũ · cần đồng bộ'
          : packAction === 'sync' ? 'Cần đồng bộ mod' : packAction === 'update' ? 'Có bản cập nhật' : 'Modpack đã sẵn sàng';

  useEffect(() => {
    if (fromServer && !loading && !loadError) void checkPack(selectedServer);
  }, [fromServer, selectedServer, loading, loadError]);

  const toggleTunnel = async () => {
    setTunnelBusy(true);
    try {
      setTunnelStatus(await ToggleTunnel());
    } catch (error) {
      showToast(`Không thể đổi trạng thái Cam Tunnel. ${error}`);
    } finally {
      setTunnelBusy(false);
    }
  };

  const runAction = async (skipConnection = false, skipUpdate = false) => {
    if (currentActivity.current || runningTitle || loading || loadError || (!fromServer && !localInstance)) return;
    setActionError(null);

    if (action === 'check' && !skipUpdate) {
      await checkPack(selectedServer);
      return;
    }
    if (action === 'account' || (skipUpdate && needsAccount)) {
      if (accounts.some((account) => account.username.toLowerCase() !== 'camplayer')) setCurrentTab('accounts');
      else setShowAccountSetup(true);
      return;
    }
    const effectiveAction = skipUpdate ? 'play' : action;
    const pending: Activity = effectiveAction === 'java'
      ? { kind: 'java', title, label: 'Đang cài Java 17…' }
      : effectiveAction === 'play'
        ? { kind: 'launch', title, label: `Đang chuẩn bị ${title}…` }
        : { kind: 'pack', title, label: effectiveAction === 'install' ? `Đang cài ${title}…` : `Đang đồng bộ ${title}…` };
    currentActivity.current = pending;
    setActivity(pending);
    setProgress(null);

    let waitingForGame = false;
    try {
      if (pending.kind === 'java') {
        setEnvSetup(await AutoSetupEnvironment());
      } else if (pending.kind === 'pack') {
        const installed = await SyncOfficialModpack(selectedServer.id);
        setInstances((previous) => [...previous.filter((instance) => instance.id !== installed.id), installed]);
        await checkPack(selectedServer);
      } else if (playableInstance) {
        if (fromServer && selectedServer.id === 'chilltown' && !skipConnection) {
          const status = await GetTunnelStatus();
          if (!status.running) {
            setActivity({ ...pending, label: 'Đang kết nối Chill Town…' });
            try {
              const connected = await StartTunnel();
              setTunnelStatus(connected);
              if (!connected.running) throw new Error('Cam Tunnel chưa kết nối.');
            } catch (error) {
              const direct = await GetServerLiveStatus().catch(() => null);
              if (!direct?.online) {
                setActionError({ title, message: String(error), offlineAvailable: true });
                return;
              }
            }
          }
        }
        setActivity(pending);
        await LaunchGame(playableInstance.id);
        waitingForGame = true;
      }
    } catch (error) {
      setActionError({ title, message: String(error) });
    } finally {
      if (!waitingForGame) finishActivity();
    }
  };

  const stopGame = async () => {
    setStopping(true);
    try {
      await KillGame();
    } catch (error) {
      setStopping(false);
      setActionError({ title: runningTitle, message: String(error) });
    }
  };

  const primaryLabel = {
    java: 'Cài Java 17',
    install: `Cài ${title}${selectedServer.isKamaron ? ' · 434 MB' : ''}`,
    update: `Cập nhật ${title}`,
    sync: `Đồng bộ ${title}`,
    account: 'Chọn tài khoản',
    play: `Chơi ${title}`,
    check: 'Kiểm tra lại modpack',
  }[action];
  const PrimaryIcon = action === 'play' ? Play : action === 'sync' ? RefreshCw : action === 'account' ? Box : Download;
  const percent = Math.floor(Math.max(0, Math.min(100, progress?.percentage ?? 0)));
  const activityMessage = progress?.message || activity?.label;
  const currentError = actionError && ((showPlay && actionError.title === title) || runningTitle) ? actionError : null;
  const dockDescription = fromServer
    ? needsJava ? 'Cần Java 17 để chạy bản này' : packStatus
    : localInstance
      ? `Minecraft ${localInstance.gameVersion} · ${localInstance.loader || 'Vanilla'} · ${Math.round(localInstance.maxMemoryMb / 1024)} GB RAM`
      : 'Chọn bản muốn chơi từ danh sách';

  return (
    <div className="launcher-shell">
      <Titlebar appVersion={appVersion} launcherUpdate={launcherUpdate} onOpenLauncherUpdate={() => setShowLauncherUpdate(true)} />
      <div className="launcher-body">
        <Sidebar currentTab={currentTab} onSelectTab={setCurrentTab} activeAccount={activeAccount}
          instanceCount={instances.length} />
        <div className="launcher-workspace">
          {loadError && (
            <div className="launcher-notice" role="alert">
              Không thể nạp tài khoản và bản cài.
              <button className="cam-button" onClick={loadLauncher}>Thử lại</button>
            </div>
          )}
          {fromServer && (
            <PlayView key={selectedServer.id} selectedServer={selectedServer} onSelectServer={setSelectedServer}
              tunnelStatus={tunnelStatus} tunnelBusy={tunnelBusy || activity?.kind === 'launch'}
              onToggleTunnel={toggleTunnel} packStatus={loading ? 'Đang nạp bản cài…' : loadError ? 'Chưa nạp được bản cài' : packStatus}
              checkingPack={pack?.status === 'checking'} onCheckPack={() => checkPack(selectedServer)}
              installed={Boolean(officialInstance)} busy={Boolean(activity)}
              needsJava={needsJava} updateResult={pack?.result} showToast={showToast} />
          )}
          {currentTab === 'instances' && (
            <InstancesView instances={instances} activeInstance={localInstance} stats={stats}
              onSelectInstance={(instance) => setLocalId(instance.id)}
              onOpenSettings={(instance) => setSettingsId(instance.id)} onRefresh={refreshData} showToast={showToast} />
          )}
          {currentTab === 'mods' && (
            <ModStoreView activeInstance={modTarget} instances={instances} onSelectInstance={(instance) => setModTargetId(instance.id)} />
          )}
          {currentTab === 'accounts' && <AccountsView accounts={accounts} activeAccount={activeAccount} onRefresh={refreshDataWithNotice} />}
          {currentTab === 'console' && <ConsoleView logs={logs} onClearLogs={() => setLogs([])} isRunning={Boolean(runningTitle)} />}
          {currentTab === 'settings' && (
            <SettingsView stats={stats} appVersion={appVersion} launcherUpdate={launcherUpdate} checkingLauncher={checkingLauncher}
              launcherCheckError={launcherCheckError}
              onCheckLauncherUpdate={() => checkLauncher()} onOpenLauncherUpdate={() => setShowLauncherUpdate(true)}
              onEnvironmentChanged={setEnvSetup} showToast={showToast} />
          )}

          {(showPlay || activity || runningTitle || currentError) && (
            <footer className="launch-dock">
              <div className="launch-target">
                <span className="launch-target-icon"><Box size={21} /></span>
                <div className="launch-target-copy">
                  <strong>{runningTitle || activity?.title || currentError?.title || title}</strong>
                  {currentError ? (
                    <span className="launch-error" role="alert" title={currentError.message}><AlertCircle size={14} /> {currentError.message}</span>
                  ) : (
                    <span role="status" title={activityMessage}>{runningTitle ? 'Minecraft đang chạy' : activityMessage || dockDescription}</span>
                  )}
                </div>
              </div>
              <div className="launch-actions">
                {(currentError || runningTitle) && currentTab !== 'console' && (
                  <button className="cam-button" onClick={() => setCurrentTab('console')}><Terminal size={17} /> Nhật ký</button>
                )}
                {runningTitle ? (
                  <button className="cam-button cam-danger" onClick={stopGame} disabled={stopping}>
                    {stopping ? <Loader2 size={18} className="animate-spin" /> : <Square size={17} />} {stopping ? 'Đang dừng…' : 'Dừng game'}
                  </button>
                ) : activity ? (
                  <div className="launch-progress" role="status">
                    <Loader2 size={18} className="animate-spin" />
                    <span>{activity.label}</span>
                    {percent > 0 && <progress max={100} value={percent} aria-label="Tiến trình" />}
                  </div>
                ) : showPlay ? (
                  <>
                    {currentError?.offlineAvailable && fromServer ? (
                      <button className="cam-button" onClick={() => runAction(true)}>Mở Minecraft ngoại tuyến</button>
                    ) : fromServer && pack?.status === 'error' && playableInstance ? (
                      <button className="cam-button" onClick={() => runAction(true, true)}>Mở Minecraft ngoại tuyến</button>
                    ) : playableInstance && (
                      <>
                        <button className="cam-button cam-icon" title="Mở thư mục bản cài" aria-label="Mở thư mục bản cài"
                          onClick={() => OpenInstanceFolder(playableInstance.id).catch((error) => showToast(String(error)))}><FolderOpen size={19} /></button>
                        <button className="cam-button cam-icon" title="Cấu hình bản cài" aria-label="Cấu hình bản cài"
                          onClick={() => setSettingsId(playableInstance.id)}><Settings size={19} /></button>
                      </>
                    )}
                    <button className="cam-button cam-primary launch-primary"
                      disabled={loading || loadError || (fromServer ? Boolean(officialInstance && (!pack || pack.status === 'checking')) || tunnelBusy : !localInstance)}
                      onClick={() => runAction()}>
                      <PrimaryIcon size={19} />
                      <span>{loading ? 'Đang nạp…' : fromServer && officialInstance && (!pack || pack.status === 'checking')
                        ? 'Đang kiểm tra modpack…' : currentError?.offlineAvailable ? 'Kết nối lại'
                          : fromServer || localInstance ? primaryLabel : 'Chọn bản cài'}</span>
                    </button>
                  </>
                ) : null}
              </div>
            </footer>
          )}
        </div>
      </div>

      {showLauncherUpdate && launcherUpdate && (
        <LauncherUpdateModal updateInfo={launcherUpdate} onClose={() => setShowLauncherUpdate(false)} progress={progress} />
      )}
      <InstanceSettingsModal instance={settingsInstance} stats={stats} isOpen={Boolean(settingsInstance)}
        onClose={() => setSettingsId(undefined)}
        onUpdated={(updated) => setInstances((previous) => previous.map((instance) => instance.id === updated.id ? updated : instance))}
        showToast={showToast} />
      <AccountSetupModal isOpen={showAccountSetup} existingAccounts={accounts} onClose={() => setShowAccountSetup(false)}
        onAccountCreated={(account) => { setActiveAccount(account); setShowAccountSetup(false); void refreshDataWithNotice(); }}
        onOpenMicrosoftLogin={() => { setShowAccountSetup(false); setCurrentTab('accounts'); }} showToast={showToast} />
      {toast && <div className="launcher-toast" role="status">{toast}</div>}
    </div>
  );
}

export default App;
