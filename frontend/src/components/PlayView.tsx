import React, { useState, useEffect } from 'react';
import { RecommendedServerItem } from '../data/servers';
import { tunnel, models } from '../../wailsjs/go/models';
import {
  GetServerLiveStatus,
  CheckOfficialServerUpdate,
  SyncOfficialChillTown,
} from '../../wailsjs/go/main/App';
import {
  Shield,
  Copy,
  Check,
  Radio,
  Sparkles,
  Users,
  RotateCw,
  RefreshCw,
  CheckCircle2,
  Box,
  Cpu,
  Coins,
  Zap,
} from 'lucide-react';

interface PlayViewProps {
  selectedServer: RecommendedServerItem;
  onSelectServer?: (server: RecommendedServerItem) => void;
  tunnelStatus?: tunnel.TunnelStatus | null;
  onToggleTunnel?: () => void;
  showToast?: (msg: string) => void;
  officialUpdate?: models.UpdateCheckResult | null;
  isSyncing?: boolean;
  onTriggerSync?: () => void;
  onUpdateStatusChanged?: (res: models.UpdateCheckResult) => void;
}

export const PlayView: React.FC<PlayViewProps> = ({
  selectedServer,
  tunnelStatus,
  onToggleTunnel,
  showToast,
  officialUpdate,
  isSyncing = false,
  onTriggerSync,
  onUpdateStatusChanged,
}) => {
  const [copiedIp, setCopiedIp] = useState(false);
  const [liveStatus, setLiveStatus] = useState<models.ServerLiveStatus | null>(null);
  const [isRefreshingStatus, setIsRefreshingStatus] = useState(false);
  const [checkingUpdate, setCheckingUpdate] = useState(false);
  const [updateResult, setUpdateResult] = useState<models.UpdateCheckResult | null>(officialUpdate || null);

  useEffect(() => {
    if (officialUpdate) {
      setUpdateResult(officialUpdate);
    }
  }, [officialUpdate]);

  // Poll live Minecraft Server List Ping (SLP)
  const fetchLiveStatus = async () => {
    try {
      const res = await GetServerLiveStatus();
      setLiveStatus(res);
    } catch (e) {
      console.warn('Failed to get server live status:', e);
    }
  };

  useEffect(() => {
    fetchLiveStatus();
    const timer = setInterval(fetchLiveStatus, 4000);
    return () => clearInterval(timer);
  }, [tunnelStatus?.running]);

  const handleManualRefresh = async () => {
    setIsRefreshingStatus(true);
    await fetchLiveStatus();
    if (showToast) showToast('Đã làm mới trạng thái máy chủ');
    setTimeout(() => setIsRefreshingStatus(false), 600);
  };

  const handleCheckUpdate = async () => {
    setCheckingUpdate(true);
    try {
      const res = await CheckOfficialServerUpdate();
      setUpdateResult(res);
      if (onUpdateStatusChanged) onUpdateStatusChanged(res);
      if (res.isDrifted) {
        if (showToast)
          showToast(
            `Mod đang bị lệch (${res.missingMods?.length || 0} thiếu, ${
              res.extraMods?.length || 0
            } thừa). Bấm Đồng Bộ để sửa.`
          );
      } else if (res.hasUpdate) {
        if (showToast) showToast(`Có bản cập nhật Chill Town ${res.latestVersion}. Bấm Cập Nhật để tải.`);
      } else {
        if (showToast) showToast('Modpack Chill Town đã chuẩn và sẵn sàng.');
      }
    } catch (err: any) {
      if (showToast) showToast(`Lỗi kiểm tra cập nhật: ${err}`);
    } finally {
      setCheckingUpdate(false);
    }
  };

  const handleSyncUpdate = async () => {
    if (onTriggerSync) {
      onTriggerSync();
      return;
    }
    setCheckingUpdate(true);
    try {
      await SyncOfficialChillTown();
      const refreshed = await CheckOfficialServerUpdate();
      setUpdateResult(refreshed);
      if (onUpdateStatusChanged) onUpdateStatusChanged(refreshed);
      if (showToast) showToast('Đã đồng bộ modpack Chill Town. Sẵn sàng chơi!');
    } catch (err: any) {
      if (showToast) showToast(`Lỗi đồng bộ Chill Town: ${err}`);
    } finally {
      setCheckingUpdate(false);
    }
  };

  const handleCopyIp = () => {
    const address = selectedServer.serverAddress || '127.0.0.1:25565';
    navigator.clipboard.writeText(address);
    setCopiedIp(true);
    if (showToast) showToast(`Đã sao chép địa chỉ IP: ${address}`);
    setTimeout(() => setCopiedIp(false), 2000);
  };

  const isServerOnline = Boolean(liveStatus?.online);
  const pingVal = isServerOnline ? liveStatus?.pingMs || selectedServer.pingMs : selectedServer.pingMs;
  const currentOnlinePlayers = isServerOnline ? liveStatus?.players : 0;
  const maxOnlinePlayers = isServerOnline ? liveStatus?.maxPlayers : 20;

  return (
    <main className="flex-1 overflow-y-auto flex flex-col select-none relative z-10 p-6 gap-5 box-border">
      {/* Top Header Row with Clear Connectivity Capsule */}
      <div className="flex items-center justify-between flex-wrap gap-4 shrink-0">
        <div className="flex items-center gap-3">
          <h1 className="text-xl md:text-2xl font-black tracking-tight text-white flex items-center gap-2">
            <span>Máy Chủ Đề Xuất</span>
          </h1>
          <span className="status-pill rounded-full font-bold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30 text-[11px] px-3 py-0.5 tracking-wide">
            Chính Thức
          </span>
        </div>

        {/* Quick Connectivity Capsule */}
        <div className="flex items-center gap-3 flex-wrap">
          {/* WireGuard P2P Toggle Pill */}
          <div
            onClick={onToggleTunnel}
            className={`glass-panel border rounded-2xl flex items-center gap-3 px-3.5 py-1.5 transition-all btn-tactile cursor-pointer ${
              tunnelStatus?.running
                ? 'border-emerald-500/40 bg-emerald-500/[0.08] text-white shadow-lg shadow-emerald-500/10'
                : 'border-white/10 bg-black/40 text-slate-300 hover:border-white/20'
            }`}
            title="Đường truyền WireGuard P2P độ trễ thấp trực tiếp tới máy chủ"
          >
            <div className="flex items-center gap-2">
              <span className="relative flex h-2.5 w-2.5">
                {tunnelStatus?.running && (
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                )}
                <span
                  className={`relative inline-flex rounded-full h-2.5 w-2.5 ${
                    tunnelStatus?.running ? 'bg-emerald-400' : 'bg-slate-500'
                  }`}
                />
              </span>
              <span className="text-xs font-bold font-mono">P2P WireGuard</span>
            </div>

            <span
              className={`status-pill px-2 py-0.5 rounded-lg text-[10px] font-bold uppercase tracking-wider ${
                tunnelStatus?.running
                  ? 'bg-emerald-500 text-slate-950'
                  : 'bg-white/10 text-slate-400'
              }`}
            >
              {tunnelStatus?.running ? 'Bật' : 'Tắt'}
            </span>
          </div>

          {/* Server Ping & Online Capsule */}
          <div
            onClick={handleManualRefresh}
            className={`glass-panel border rounded-2xl flex items-center gap-2.5 px-3.5 py-1.5 text-xs font-mono transition-all btn-tactile cursor-pointer ${
              isServerOnline
                ? 'border-emerald-500/30 bg-emerald-500/[0.05] text-emerald-300'
                : 'border-white/10 bg-black/40 text-slate-400'
            }`}
            title="Bấm để kiểm tra lại ping và trạng thái máy chủ"
          >
            <span className="relative flex h-2 w-2">
              {isServerOnline && (
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
              )}
              <span
                className={`relative inline-flex rounded-full h-2 w-2 ${
                  isServerOnline ? 'bg-emerald-400' : 'bg-rose-500'
                }`}
              />
            </span>
            <span className="font-bold">{isServerOnline ? `${pingVal}ms` : 'Ngoại Tuyến'}</span>
            <span className="text-slate-500">•</span>
            <span className="text-slate-300 font-sans text-[11px]">
              {currentOnlinePlayers}/{maxOnlinePlayers} Online
            </span>
            <RefreshCw
              className={`w-3 h-3 text-slate-400 ml-0.5 ${
                isRefreshingStatus ? 'animate-spin text-emerald-400' : ''
              }`}
            />
          </div>

          {/* Copy IP Capsule */}
          <button
            onClick={handleCopyIp}
            className="glass-panel border border-white/10 hover:border-emerald-500/35 rounded-2xl flex items-center gap-2 px-3 py-1.5 text-xs font-mono text-slate-300 hover:text-emerald-300 transition-all btn-tactile cursor-pointer bg-black/40"
            title="Sao chép địa chỉ IP máy chủ"
          >
            <span className="text-slate-500">IP:</span>
            <span className="font-bold text-emerald-300">
              {selectedServer.serverAddress || '127.0.0.1:25565'}
            </span>
            {copiedIp ? (
              <Check className="w-3.5 h-3.5 text-emerald-400" />
            ) : (
              <Copy className="w-3.5 h-3.5 text-slate-400" />
            )}
          </button>
        </div>
      </div>

      {/* Hero Showcase Card - Clean, Cinematic & Uncluttered */}
      <section className="relative rounded-3xl overflow-hidden border border-white/10 flex flex-col justify-between shadow-2xl bg-slate-950/80 group flex-1 p-8 gap-6">
        {/* Cinematic Backdrop with Subtle Zoom */}
        <div className="absolute inset-0 pointer-events-none z-0 overflow-hidden">
          <img
            src={selectedServer.bannerUrl}
            alt={selectedServer.title}
            className="w-full h-full object-cover opacity-20 group-hover:scale-105 transition-transform duration-1000 ease-out"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-[#0a0d14] via-[#0a0d14]/70 to-transparent" />
          <div className="absolute inset-0 bg-gradient-to-r from-[#0a0d14] via-[#0a0d14]/60 to-transparent" />
        </div>

        {/* Top Badges */}
        <div className="relative z-10 flex items-center justify-between flex-wrap gap-3">
          <div className="flex items-center gap-2.5">
            <span className="status-pill rounded-lg bg-emerald-500/20 text-emerald-300 border border-emerald-500/35 font-extrabold text-[11px] px-3 py-1 tracking-wider uppercase flex items-center gap-1.5">
              <Shield className="w-3.5 h-3.5 text-emerald-400" />
              <span>CHILL TOWN COMMUNITY</span>
            </span>

            <span className="status-pill rounded-lg bg-white/[0.06] text-slate-300 border border-white/10 font-mono text-[11px] px-3 py-1">
              Forge 1.20.1 • 246 Mods
            </span>
          </div>

          <span className="text-xs font-mono text-slate-400 flex items-center gap-1.5">
            <Radio className="w-3.5 h-3.5 text-emerald-400" />
            <span>P2P Direct Tunnel Ready</span>
          </span>
        </div>

        {/* Focal Title & Tagline */}
        <div className="relative z-10 flex flex-col gap-3 max-w-3xl my-auto">
          <h2 className="text-3xl md:text-5xl font-black text-white tracking-tight leading-tight flex items-center gap-3">
            <span>{selectedServer.title}</span>
          </h2>

          <p className="text-sm md:text-base text-slate-300 leading-relaxed max-w-2xl">
            Thế giới sinh tồn ẩm thực kết hợp cơ giới hóa nông nghiệp cao cấp — Tích hợp hệ thống kinh tế SDM Shop, chuỗi nhiệm vụ và đồ hoạ shader tối ưu.
          </p>

          {/* Clean Tag Pills */}
          <div className="flex flex-wrap gap-2 pt-1">
            <span className="status-pill bg-white/[0.05] hover:bg-white/[0.08] text-slate-200 border border-white/10 rounded-lg text-xs font-semibold px-3 py-1 flex items-center gap-1.5">
              <Zap className="w-3.5 h-3.5 text-amber-400" />
              <span>Tối Ưu FPS & Shader</span>
            </span>
            <span className="status-pill bg-white/[0.05] hover:bg-white/[0.08] text-slate-200 border border-white/10 rounded-lg text-xs font-semibold px-3 py-1 flex items-center gap-1.5">
              <Box className="w-3.5 h-3.5 text-emerald-400" />
              <span>Farmer's Delight & Cơ Giới</span>
            </span>
            <span className="status-pill bg-white/[0.05] hover:bg-white/[0.08] text-slate-200 border border-white/10 rounded-lg text-xs font-semibold px-3 py-1 flex items-center gap-1.5">
              <Coins className="w-3.5 h-3.5 text-cyan-400" />
              <span>SDM Shop Kinh Tế</span>
            </span>
          </div>
        </div>

        {/* 4-Column Spec Highlights Strip (Sleek, Compact, No Clutter) */}
        <div className="relative z-10 grid grid-cols-2 md:grid-cols-4 gap-3 pt-2">
          <div className="glass-card rounded-2xl px-4 py-3 flex items-center gap-3 border border-white/10 hover:border-emerald-500/30 transition-all card-interactive">
            <div className="w-9 h-9 rounded-xl bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center shrink-0">
              <Box className="w-4 h-4 text-emerald-400" />
            </div>
            <div className="flex flex-col min-w-0">
              <span className="text-xs font-bold text-white tracking-wide truncate">246 Mods Nạp Sẵn</span>
              <span className="text-[10.5px] text-slate-400 truncate">Tự động đồng bộ KubeJS</span>
            </div>
          </div>

          <div className="glass-card rounded-2xl px-4 py-3 flex items-center gap-3 border border-white/10 hover:border-purple-500/30 transition-all card-interactive">
            <div className="w-9 h-9 rounded-xl bg-purple-500/15 border border-purple-500/30 flex items-center justify-center shrink-0">
              <Cpu className="w-4 h-4 text-purple-400" />
            </div>
            <div className="flex flex-col min-w-0">
              <span className="text-xs font-bold text-white tracking-wide truncate">Forge 1.20.1</span>
              <span className="text-[10.5px] text-slate-400 truncate">Java 17 LTS Adoptium</span>
            </div>
          </div>

          <div className="glass-card rounded-2xl px-4 py-3 flex items-center gap-3 border border-white/10 hover:border-amber-500/30 transition-all card-interactive">
            <div className="w-9 h-9 rounded-xl bg-amber-500/15 border border-amber-500/30 flex items-center justify-center shrink-0">
              <Coins className="w-4 h-4 text-amber-400" />
            </div>
            <div className="flex flex-col min-w-0">
              <span className="text-xs font-bold text-white tracking-wide truncate">Kinh Tế & Quests</span>
              <span className="text-[10.5px] text-slate-400 truncate">SDM Shop, FTB Quests</span>
            </div>
          </div>

          <div className="glass-card rounded-2xl px-4 py-3 flex items-center gap-3 border border-white/10 hover:border-cyan-500/30 transition-all card-interactive">
            <div className="w-9 h-9 rounded-xl bg-cyan-500/15 border border-cyan-500/30 flex items-center justify-center shrink-0">
              <Radio className="w-4 h-4 text-cyan-400" />
            </div>
            <div className="flex flex-col min-w-0">
              <span className="text-xs font-bold text-white tracking-wide truncate">P2P WireGuard</span>
              <span className="text-[10.5px] text-slate-400 truncate">Độ trễ thấp nội địa (~14ms)</span>
            </div>
          </div>
        </div>

        {/* Server Sub-Bar: Clean Modpack Status & Drift Handling */}
        <div className="relative z-10 flex items-center justify-between flex-wrap gap-4 pt-4 border-t border-white/10">
          {/* Modpack Sync Status */}
          <div className="flex items-center gap-3 flex-wrap">
            {updateResult?.isDrifted ? (
              <div className="flex items-center gap-2.5 flex-wrap">
                <div className="flex items-center gap-2 bg-amber-500/10 border border-amber-500/30 rounded-xl px-3 py-1.5 text-xs text-amber-200">
                  <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
                  <span className="font-semibold">
                    Mod đang bị lệch ({updateResult.missingMods?.length || 0} thiếu, {updateResult.extraMods?.length || 0} thừa). Bạn có muốn đồng bộ?
                  </span>
                </div>
                <button
                  onClick={handleSyncUpdate}
                  disabled={checkingUpdate || isSyncing}
                  className="status-pill px-4 py-1.5 rounded-xl text-xs font-black transition-all cursor-pointer flex items-center gap-2 bg-gradient-to-r from-amber-500 to-emerald-400 hover:from-amber-400 hover:to-emerald-300 text-slate-950 shadow-md shadow-amber-500/20 btn-tactile"
                >
                  <RotateCw className={`w-3.5 h-3.5 ${checkingUpdate || isSyncing ? 'animate-spin' : ''}`} />
                  <span>{checkingUpdate || isSyncing ? 'Đang Đồng Bộ...' : 'Đồng Bộ Mod Ngay'}</span>
                </button>
              </div>
            ) : updateResult?.hasUpdate ? (
              <div className="flex items-center gap-2.5 flex-wrap">
                <div className="flex items-center gap-2 bg-black/40 border border-white/10 rounded-xl px-3 py-1.5 text-xs text-slate-300">
                  <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                  <span className="text-emerald-300 font-semibold">
                    Có bản cập nhật mới v{updateResult.latestVersion}
                  </span>
                </div>
                <button
                  onClick={handleSyncUpdate}
                  disabled={checkingUpdate || isSyncing}
                  className="status-pill px-3.5 py-1.5 rounded-xl text-xs font-black transition-all cursor-pointer flex items-center gap-2 bg-emerald-500 hover:bg-emerald-400 text-slate-950 shadow-md shadow-emerald-500/20 btn-tactile"
                >
                  <CheckCircle2 className={`w-3.5 h-3.5 ${checkingUpdate || isSyncing ? 'animate-spin' : ''}`} />
                  <span>{checkingUpdate || isSyncing ? 'Đang Cập Nhật...' : 'Cập Nhật Modpack'}</span>
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2 bg-black/40 border border-white/10 rounded-xl px-3 py-1.5 text-xs text-slate-300">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                <span className="text-slate-300">Modpack Chill Town đã chuẩn và sẵn sàng</span>
              </div>
            )}

            <button
              onClick={handleCheckUpdate}
              disabled={checkingUpdate || isSyncing}
              className="status-pill px-3 py-1.5 rounded-xl text-xs font-semibold transition-all cursor-pointer flex items-center gap-1.5 bg-white/[0.04] hover:bg-white/[0.08] text-slate-400 hover:text-white border border-white/10 btn-tactile"
              title="So sánh bản cài hiện tại với danh sách mod chuẩn máy chủ"
            >
              <RotateCw className={`w-3.5 h-3.5 ${checkingUpdate ? 'animate-spin' : ''}`} />
              <span>{checkingUpdate ? 'Đang kiểm tra...' : 'Kiểm tra cập nhật'}</span>
            </button>
          </div>

          {/* Action Guidance */}
          <div className="text-xs text-slate-400 font-medium hidden sm:flex items-center gap-1.5">
            {updateResult?.isDrifted ? (
              <span>Bấm <strong className="text-amber-400 font-bold uppercase">Đồng Bộ Mod</strong> để chuẩn hoá lại modpack.</span>
            ) : updateResult?.hasUpdate ? (
              <span>Bấm <strong className="text-emerald-400 font-bold uppercase">Cập Nhật</strong> ở thanh dưới để tải bản mới.</span>
            ) : (
              <span>Bấm nút <strong className="text-emerald-400 font-black uppercase">CHƠI</strong> ở góc dưới để bắt đầu.</span>
            )}
          </div>
        </div>
      </section>
    </main>
  );
};
