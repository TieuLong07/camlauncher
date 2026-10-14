import React, { useState } from 'react';
import {
  Sparkles,
  Download,
  X,
  CheckCircle2,
  RefreshCw,
  Zap,
  ShieldCheck,
} from 'lucide-react';
import { models } from '../../wailsjs/go/models';
import { ApplyLauncherUpdate } from '../../wailsjs/go/main/App';

interface LauncherUpdateModalProps {
  updateInfo: models.LauncherUpdateInfo | null;
  onClose: () => void;
  progress: any | null;
}

export const LauncherUpdateModal: React.FC<LauncherUpdateModalProps> = ({
  updateInfo,
  onClose,
  progress,
}) => {
  const [updating, setUpdating] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  if (!updateInfo || !updateInfo.hasUpdate) return null;

  const handleApply = async () => {
    setUpdating(true);
    setErrorMsg('');
    try {
      await ApplyLauncherUpdate(updateInfo.downloadUrl, updateInfo.sha256 || '');
      // Backend will hot-replace the exe and restart automatically
    } catch (err: any) {
      setErrorMsg(err.toString());
      setUpdating(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-4">
      <div className="glass-panel w-full max-w-lg p-6 space-y-5 border border-cyan-500/40 shadow-2xl relative animate-in fade-in zoom-in-95 duration-200">
        {/* Header */}
        <div className="flex items-start justify-between border-b border-white/10 pb-4">
          <div className="flex items-center gap-3">
            <div className="w-11 h-11 rounded-2xl bg-gradient-to-tr from-cyan-500 to-emerald-400 text-black flex items-center justify-center shadow-lg shadow-cyan-500/20 font-black">
              <Sparkles className="w-6 h-6 fill-black" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-black text-white">Có Bản Cập Nhật CamLauncher Mới!</h2>
                <span className="px-2 py-0.5 rounded-full text-[10px] font-extrabold uppercase bg-cyan-500/20 text-cyan-300 border border-cyan-500/40">
                  v{updateInfo.latestVersion}
                </span>
              </div>
              <p className="text-xs text-slate-400 mt-0.5">
                Phiên bản hiện tại: <span className="font-mono text-slate-300">v{updateInfo.currentVersion}</span>
              </p>
            </div>
          </div>

          {!updating && (
            <button
              onClick={onClose}
              className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-white/10 cursor-pointer"
            >
              <X className="w-5 h-5" />
            </button>
          )}
        </div>

        {/* Feature highlight */}
        <div className="flex items-center gap-3 px-3.5 py-2 rounded-xl bg-cyan-950/40 border border-cyan-500/30 text-xs text-cyan-200">
          <Zap className="w-4 h-4 text-cyan-400 shrink-0" />
          <span>
            <strong>Cập nhật không cần cài lại:</strong> Launcher tự động thay thế file chạy và khởi động lại tức thì trong 2 giây.
          </span>
        </div>

        {/* Changelog box */}
        <div className="space-y-2">
          <div className="flex items-center justify-between text-xs text-slate-300 font-bold uppercase tracking-wider">
            <span>Nội dung cập nhật {updateInfo.title ? `(${updateInfo.title})` : ''}:</span>
            {updateInfo.releaseDate && (
              <span className="text-[11px] text-slate-500 font-normal font-mono">
                {updateInfo.releaseDate}
              </span>
            )}
          </div>
          <div className="bg-black/50 rounded-xl p-3.5 border border-white/5 text-xs text-slate-300 space-y-1.5 max-h-48 overflow-y-auto leading-relaxed whitespace-pre-line font-sans">
            {updateInfo.changelog || '• Tối ưu hiệu năng khởi chạy game\n• Nâng cấp hệ thống đồng bộ modpack\n• Sửa một số lỗi giao diện'}
          </div>
        </div>

        {/* Error message if any */}
        {errorMsg && (
          <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs">
            Lỗi cập nhật: {errorMsg}
          </div>
        )}

        {/* Progress bar during download */}
        {updating && progress && (
          <div className="bg-black/40 rounded-xl p-3.5 border border-cyan-500/30 space-y-2">
            <div className="flex items-center justify-between text-xs">
              <span className="text-cyan-300 font-semibold">{progress.message || 'Đang tải bản cập nhật...'}</span>
              <span className="text-white font-mono">{Math.round(progress.percentage || 0)}%</span>
            </div>
            <div className="w-full h-2 bg-black/60 rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-cyan-400 to-emerald-400 transition-all duration-200"
                style={{ width: `${progress.percentage || 0}%` }}
              />
            </div>
            <div className="flex items-center justify-between text-[11px] text-slate-400">
              <span>CamLauncher.exe</span>
              {progress.speedKbps > 0 && (
                <span className="font-mono text-cyan-400">{Math.round(progress.speedKbps)} KB/s</span>
              )}
            </div>
          </div>
        )}

        {/* Footer actions */}
        <div className="flex items-center justify-end gap-3 pt-2 border-t border-white/10">
          {!updating && (
            <button
              onClick={onClose}
              className="btn-secondary px-4 py-2 text-xs font-semibold cursor-pointer"
            >
              Để Sau
            </button>
          )}

          <button
            onClick={handleApply}
            disabled={updating}
            className="btn-primary px-6 py-2.5 text-xs font-bold shadow-lg shadow-emerald-500/20 cursor-pointer flex items-center gap-2"
          >
            {updating ? (
              <>
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                <span>Đang Cập Nhật & Khởi Động Lại...</span>
              </>
            ) : (
              <>
                <Download className="w-4 h-4" />
                <span>Cập Nhật & Khởi Động Lại Ngay</span>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
