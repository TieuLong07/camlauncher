import React from 'react';
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime';
import { models } from '../../wailsjs/go/models';

interface TitlebarProps {
  stats: models.SystemStats | null;
  activeAccount: models.Account | null;
  launcherUpdate: models.LauncherUpdateInfo | null;
  onOpenAccounts: () => void;
  onOpenLauncherUpdate: () => void;
  isRunning?: boolean;
}

const formatCpu = (model: string | undefined): string => {
  if (!model) return 'CPU';
  const s = model
    .replace(/\(R\)|\(TM\)/gi, '')
    .replace(/@.*$/g, '')
    .replace(/CPU/gi, '')
    .replace(/Processor/gi, '')
    .replace(/with Radeon.*/gi, '')
    .replace(/\d+-Core/gi, '')
    .replace(/Six-Core|Eight-Core|Quad-Core/gi, '')
    .trim();

  // Highlight Intel Core (e.g. i5-12400F, i7-13700K)
  const intelMatch = s.match(/(i[3579]-\d+[A-Z]*)/i);
  if (intelMatch) return intelMatch[1].toUpperCase();

  // Highlight AMD Ryzen (e.g. Ryzen 5 5600X)
  const ryzenMatch = s.match(/Ryzen\s*([3579]\s*\d+[A-Z]*)/i);
  if (ryzenMatch) return `Ryzen ${ryzenMatch[1]}`;

  return s.length > 18 ? s.slice(0, 18).trim() : s;
};

const formatGpu = (model: string | undefined): string => {
  if (!model) return 'GPU';
  const s = model
    .replace(/NVIDIA\s+GeForce\s+/i, '')
    .replace(/NVIDIA\s+/i, '')
    .replace(/AMD\s+Radeon\s+/i, '')
    .replace(/Laptop\s+GPU/i, '')
    .replace(/Graphics/i, 'GPU')
    .trim();

  return s.length > 20 ? s.slice(0, 20).trim() : s;
};

export const Titlebar: React.FC<TitlebarProps> = ({
  stats,
  launcherUpdate,
  onOpenLauncherUpdate,
  isRunning,
}) => {
  return (
    <header
      className="h-14 flex items-center justify-between border-b border-white/5 bg-[#0a0d14]/70 backdrop-blur-xl shrink-0 wails-drag z-30 select-none"
      style={{ padding: '0 18px' }}
    >
      {/* Left: Brand */}
      <div className="flex items-center" style={{ gap: '14px' }}>
        <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-emerald-400 to-teal-600 flex items-center justify-center shadow-lg shadow-emerald-500/20 text-slate-950 font-black text-sm tracking-tighter">
          Ca
        </div>
        <div className="flex items-center" style={{ gap: '10px' }}>
          <span className="font-bold tracking-tight text-white text-sm">CamLauncher</span>
          <span
            className="status-pill font-semibold rounded-full bg-amber-500/10 text-amber-200 border border-amber-500/20 font-mono"
            style={{ height: '24px', padding: '0 10px', fontSize: '10.5px' }}
          >
            ALPHA 0.1.4
          </span>
          {launcherUpdate?.hasUpdate && (
            <button
              onClick={onOpenLauncherUpdate}
              className="wails-no-drag status-pill rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 font-semibold animate-pulse hover:bg-emerald-500/30 transition-all cursor-pointer"
              title="Có phiên bản launcher mới!"
              style={{ gap: '6px', height: '24px', padding: '0 12px', fontSize: '10.5px' }}
            >
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
              <span>Cập nhật v{launcherUpdate.latestVersion}</span>
            </button>
          )}
        </div>
      </div>

      {/* Center: Hardware Specs Flex Bar (Gọn gàng đẹp chuẩn khoe cấu hình) */}
      {stats ? (
        <div
          className="hidden md:flex items-center status-pill rounded-full bg-white/[0.035] hover:bg-white/[0.06] border border-white/10 hover:border-emerald-500/35 transition-all select-none shadow-sm cursor-default"
          style={{ height: '34px', padding: '0 18px', gap: '12px' }}
          title={`Thông số phần cứng máy tính:\n• CPU: ${stats.cpuModel} (${stats.cpuCores} Cores Logic)\n• GPU: ${stats.gpuModel || 'Card tích hợp'}\n• RAM: ${Math.round(stats.totalRamMb / 1024)}GB (Còn trống: ${Math.round(stats.freeRamMb / 1024)}GB)`}
        >
          {/* CPU Metric */}
          <div className="flex items-center" style={{ gap: '6px' }}>
            <svg className="w-3.5 h-3.5 text-cyan-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <rect x="4" y="4" width="16" height="16" rx="2" strokeWidth="2" />
              <rect x="9" y="9" width="6" height="6" strokeWidth="2" />
              <path strokeLinecap="round" strokeWidth="2" d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 15h3M1 9h3M1 15h3" />
            </svg>
            <span className="font-bold text-slate-200 text-xs tracking-tight">
              {formatCpu(stats.cpuModel)}
            </span>
            <span className="text-[10px] text-cyan-400/80 font-mono font-semibold">
              {stats.cpuCores}C
            </span>
          </div>

          <span className="text-white/15 text-xs">|</span>

          {/* GPU Metric */}
          <div className="flex items-center" style={{ gap: '6px' }}>
            <svg className="w-3.5 h-3.5 text-emerald-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
            <span className="font-bold text-slate-200 text-xs tracking-tight truncate max-w-[140px]">
              {formatGpu(stats.gpuModel)}
            </span>
          </div>

          <span className="text-white/15 text-xs">|</span>

          {/* RAM Metric */}
          <div className="flex items-center" style={{ gap: '6px' }}>
            <svg className="w-3.5 h-3.5 text-amber-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            <span className="font-bold font-mono text-slate-200 text-xs tracking-tight">
              {Math.round(stats.totalRamMb / 1024)}GB
            </span>
            <span className="text-[10px] text-amber-400/90 font-mono font-semibold">
              RAM
            </span>
          </div>
        </div>
      ) : null}

      {/* Right: Window Controls */}
      <div className="flex items-center gap-1 pl-2 text-slate-400 wails-no-drag" style={{ gap: '4px' }}>
        {isRunning && (
          <span className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-emerald-500/10 text-emerald-400 text-[11px] font-medium border border-emerald-500/20 mr-2">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
            Minecraft đang chạy
          </span>
        )}
        <button
          onClick={() => WindowMinimise()}
          title="Thu nhỏ"
          className="w-7 h-7 rounded-lg hover:bg-white/5 flex items-center justify-center hover:text-white transition-colors cursor-pointer"
        >
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M20 12H4" />
          </svg>
        </button>
        <button
          onClick={() => WindowToggleMaximise()}
          title="Phóng to"
          className="w-7 h-7 rounded-lg hover:bg-white/5 flex items-center justify-center hover:text-white transition-colors cursor-pointer"
        >
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <rect width="16" height="16" x="4" y="4" rx="2" strokeWidth="2" />
          </svg>
        </button>
        <button
          onClick={() => Quit()}
          title="Đóng launcher"
          className="w-7 h-7 rounded-lg hover:bg-rose-500/20 hover:text-rose-400 flex items-center justify-center transition-colors cursor-pointer"
        >
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </header>
  );
};
