import React, { useState, useEffect } from 'react';
import {
  RecommendedServerItem,
} from '../data/servers';
import { tunnel, models } from '../../wailsjs/go/models';
import {
  GetServerLiveStatus,
  CheckOfficialServerUpdate,
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
  AlertCircle,
  Wifi,
  WifiOff,
} from 'lucide-react';

interface PlayViewProps {
  selectedServer: RecommendedServerItem;
  onSelectServer?: (server: RecommendedServerItem) => void;
  tunnelStatus?: tunnel.TunnelStatus | null;
  onToggleTunnel?: () => void;
  showToast?: (msg: string) => void;
}

export const PlayView: React.FC<PlayViewProps> = ({
  selectedServer,
  tunnelStatus,
  onToggleTunnel,
  showToast,
}) => {
  const [copiedIp, setCopiedIp] = useState(false);
  const [liveStatus, setLiveStatus] = useState<models.ServerLiveStatus | null>(null);
  const [isRefreshingStatus, setIsRefreshingStatus] = useState(false);
  const [checkingUpdate, setCheckingUpdate] = useState(false);
  const [updateResult, setUpdateResult] = useState<models.UpdateCheckResult | null>(null);

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
      if (res.hasUpdate) {
        if (showToast) showToast(`Có bản cập nhật mới (${res.latestVersion}) cho máy chủ!`);
      } else {
        if (showToast) showToast('Bản cài đặt Chill Town đã đồng bộ đầy đủ các mod mới nhất (v1.5.1)!');
      }
    } catch (err: any) {
      if (showToast) showToast(`Lỗi kiểm tra cập nhật: ${err}`);
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
    <main
      className="flex-1 overflow-y-auto flex flex-col select-none relative z-10"
      style={{ padding: '22px 24px 28px 24px', gap: '20px', boxSizing: 'border-box' }}
    >
      {/* Top Header Row */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div className="flex items-center" style={{ gap: '12px' }}>
          <h1 className="text-xl md:text-2xl font-black tracking-tight text-white flex items-center gap-2">
            <span>Máy Chủ Đề Xuất</span>
          </h1>
          <span
            className="status-pill rounded-full font-bold bg-emerald-500/15 text-emerald-300 border border-emerald-500/25 tracking-wide"
            style={{ height: '24px', padding: '0 12px', fontSize: '10.5px' }}
          >
            Chính thức
          </span>
        </div>

        {/* Cam P2P Tunnel Status Pill */}
        <div
          onClick={onToggleTunnel}
          className={`glass-panel border rounded-2xl flex items-center transition-all cursor-pointer select-none ${
            tunnelStatus?.running
              ? 'border-emerald-500/40 bg-emerald-500/[0.08] hover:bg-emerald-500/[0.12] shadow-lg shadow-emerald-500/10'
              : 'border-white/10 bg-black/40 hover:bg-white/[0.04]'
          }`}
          style={{ padding: '8px 16px', gap: '14px' }}
          title="Bấm để bật/tắt đường truyền WireGuard P2P tới máy chủ"
        >
          <div className="flex items-center gap-2.5">
            <span className="relative flex h-3 w-3">
              {tunnelStatus?.running && (
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
              )}
              <span
                className={`relative inline-flex rounded-full h-3 w-3 ${
                  tunnelStatus?.running ? 'bg-emerald-400' : 'bg-slate-500'
                }`}
              />
            </span>
            <div className="flex flex-col">
              <span className="text-xs font-bold text-white tracking-wide">
                Cam Tunnel (P2P): {tunnelStatus?.running ? 'ĐANG BẬT' : 'TẮT'}
              </span>
              <span className="text-[10px] text-slate-400 font-mono">
                {tunnelStatus?.running
                  ? `127.0.0.1:${tunnelStatus.localPort || 25565} • Tự động kết nối bảo mật`
                  : 'Bấm để kích hoạt đường truyền P2P'}
              </span>
            </div>
          </div>

          <button
            type="button"
            className={`status-pill px-3 py-1 rounded-xl font-bold text-[11px] transition-colors cursor-pointer ${
              tunnelStatus?.running
                ? 'bg-emerald-500 text-slate-950 shadow-sm'
                : 'bg-white/10 text-slate-300 hover:text-white'
            }`}
          >
            {tunnelStatus?.running ? 'Bật' : 'Tắt'}
          </button>
        </div>
      </div>

      {/* Hero Showcase Card - Full Width Spotlight */}
      <section
        className="relative rounded-3xl overflow-hidden border border-white/10 flex flex-col justify-between shadow-2xl bg-slate-950/80 group flex-1"
        style={{ minHeight: '480px', padding: '30px 34px', gap: '22px' }}
      >
        {/* Background Art with Cinematic Gradients */}
        <div className="absolute inset-0 pointer-events-none z-0 overflow-hidden">
          <img
            src={selectedServer.bannerUrl}
            alt={selectedServer.title}
            className="w-full h-full object-cover opacity-25 group-hover:scale-105 transition-transform duration-1000 ease-out"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-[#0a0d14] via-[#0a0d14]/75 to-transparent" />
          <div className="absolute inset-0 bg-gradient-to-r from-[#0a0d14] via-[#0a0d14]/65 to-transparent" />
        </div>

        {/* Top Badges Row */}
        <div className="relative z-10 flex items-center justify-between flex-wrap gap-3">
          <div className="flex items-center flex-wrap gap-2.5">
            <span className="status-pill rounded-lg bg-emerald-500/20 text-emerald-300 border border-emerald-500/35 font-bold text-[11px] px-3 py-1 tracking-wider uppercase flex items-center gap-1.5">
              <Shield className="w-3.5 h-3.5 text-emerald-400" />
              <span>CHILL TOWN HOST CHÍNH THỨC</span>
            </span>

            <span className="status-pill rounded-lg bg-blue-500/15 text-blue-300 border border-blue-500/30 font-mono font-bold text-[11px] px-3 py-1 tracking-wide uppercase flex items-center gap-1.5">
              <Radio className="w-3.5 h-3.5 text-blue-400" />
              <span>P2P TUNNEL READY</span>
            </span>

            <span className="status-pill rounded-lg bg-white/[0.06] text-slate-300 border border-white/10 font-mono text-[11px] px-3 py-1">
              {liveStatus?.version || selectedServer.version} Tối ưu FPS
            </span>
          </div>

          <div className="flex items-center gap-2">
            {/* Live Server Ping & Online Indicator */}
            <div
              onClick={handleManualRefresh}
              className={`status-pill font-mono px-3 py-1.5 rounded-xl text-xs backdrop-blur-md flex items-center gap-2 shadow-sm cursor-pointer border transition-all ${
                isServerOnline
                  ? 'bg-black/50 text-emerald-300 border-emerald-500/30 hover:border-emerald-500/50'
                  : 'bg-black/50 text-slate-400 border-white/10 hover:border-white/20'
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

              <span className="font-bold">
                {isServerOnline ? `${pingVal}ms` : 'NGOẠI TUYẾN'}
              </span>

              <span className="text-slate-400 font-sans text-[10.5px]">
                ({isServerOnline ? 'ONLINE' : 'OFFLINE'})
              </span>

              <RefreshCw className={`w-3 h-3 text-slate-400 ml-0.5 ${isRefreshingStatus ? 'animate-spin' : ''}`} />
            </div>
          </div>
        </div>

        {/* Center Main Info */}
        <div className="relative z-10 flex flex-col gap-3 max-w-3xl my-2">
          <span className="text-xs font-extrabold tracking-widest uppercase text-emerald-400 flex items-center gap-1.5">
            <Sparkles className="w-3.5 h-3.5" />
            <span>MÁY CHỦ CHÍNH THỨC CỘNG ĐỒNG CHILL TOWN</span>
          </span>

          <h2 className="text-3xl md:text-4xl lg:text-5xl font-black text-white tracking-tight leading-tight">
            {selectedServer.title}
          </h2>

          <p className="text-sm md:text-base text-slate-300 leading-relaxed max-w-2xl">
            {selectedServer.description}
          </p>

          <div className="flex flex-wrap gap-2 pt-1">
            {selectedServer.tags.map((t) => (
              <span
                key={t}
                className="status-pill bg-white/[0.05] hover:bg-white/[0.08] text-slate-200 border border-white/10 rounded-lg text-[11.5px] font-medium px-3 py-1"
              >
                {t}
              </span>
            ))}
          </div>
        </div>

        {/* 4 Feature Highlights Grid */}
        <div className="relative z-10 grid grid-cols-2 md:grid-cols-4 gap-3.5 pt-2">
          {selectedServer.features.map((feat, idx) => (
            <div
              key={idx}
              className="glass-panel border border-white/10 hover:border-emerald-500/30 rounded-2xl p-3.5 bg-black/40 hover:bg-white/[0.04] transition-all flex flex-col gap-1.5"
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold text-white tracking-wide">{feat.title}</span>
                <span className="text-[9.5px] font-mono font-bold px-1.5 py-0.5 rounded bg-emerald-500/15 text-emerald-300 border border-emerald-500/25">
                  {feat.badge}
                </span>
              </div>
              <span className="text-[11px] text-slate-400 leading-snug">{feat.desc}</span>
            </div>
          ))}
        </div>

        {/* Server Sub-Bar: Online Players + Copy IP + Update Check Action */}
        <div className="relative z-10 flex items-center justify-between flex-wrap gap-4 pt-4 border-t border-white/10">
          <div className="flex items-center gap-4 flex-wrap">
            {/* Online Count */}
            <div className="flex items-center gap-2 text-xs font-semibold text-slate-300">
              <Users className="w-4 h-4 text-emerald-400" />
              <span>
                Người chơi:{' '}
                <strong className={`font-mono ${isServerOnline ? 'text-white' : 'text-slate-400'}`}>
                  {currentOnlinePlayers}
                </strong>{' '}
                / {maxOnlinePlayers}
              </span>
            </div>

            {/* IP Box */}
            <div className="flex items-center gap-2 bg-black/50 border border-white/10 rounded-xl px-3 py-1.5 text-xs">
              <span className="text-slate-400 font-mono text-[11px]">IP:</span>
              <span className="font-mono text-emerald-300 font-bold tracking-wide">
                {selectedServer.serverAddress || '127.0.0.1:25565'}
              </span>
              <button
                onClick={handleCopyIp}
                className="ml-1 p-1 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition cursor-pointer"
                title="Sao chép địa chỉ IP"
              >
                {copiedIp ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>

          {/* Right Action: Smart Sync / Check Update */}
          <div className="flex items-center gap-2.5">
            <button
              onClick={handleCheckUpdate}
              disabled={checkingUpdate}
              className="status-pill px-3.5 py-2 rounded-xl text-xs font-bold transition-all cursor-pointer flex items-center gap-2 bg-white/[0.04] hover:bg-emerald-500/15 text-slate-300 hover:text-emerald-300 border border-white/10 hover:border-emerald-500/35"
              title="Kiểm tra đồng bộ modpack và tài nguyên mới nhất từ máy chủ"
            >
              <RotateCw className={`w-3.5 h-3.5 text-emerald-400 ${checkingUpdate ? 'animate-spin' : ''}`} />
              <span>{checkingUpdate ? 'Đang Kiểm Tra...' : 'Kiểm Tra Cập Nhật Mod'}</span>
            </button>
          </div>
        </div>
      </section>
    </main>
  );
};
