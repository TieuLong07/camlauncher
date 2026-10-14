import React, { useState, useEffect } from 'react';
import {
  Settings,
  Cpu,
  HardDrive,
  FolderOpen,
  Coffee,
  Check,
  Shield,
  Zap,
  Download,
  Loader2,
  CheckCircle2,
  AlertTriangle,
  FileSearch,
  Sparkles,
} from 'lucide-react';
import { models } from '../../wailsjs/go/models';
import {
  GetSettings,
  UpdateSettings,
  OpenFolder,
  CheckEnvironmentSetup,
  InstallJavaRuntime,
  SetDefaultJavaPath,
  BrowseJavaExecutable,
} from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

interface SettingsViewProps {
  stats: models.SystemStats | null;
  launcherUpdate: models.LauncherUpdateInfo | null;
  onCheckLauncherUpdate: () => void;
  onOpenLauncherUpdate: () => void;
  showToast?: (msg: string) => void;
}

export const SettingsView: React.FC<SettingsViewProps> = ({
  stats,
  launcherUpdate,
  onCheckLauncherUpdate,
  onOpenLauncherUpdate,
  showToast,
}) => {
  const [settings, setSettings] = useState<models.AppSettings | null>(null);
  const [saved, setSaved] = useState(false);
  const [setupStatus, setSetupStatus] = useState<models.EnvironmentSetupStatus | null>(null);
  const [loadingEnv, setLoadingEnv] = useState(true);
  const [installingVer, setInstallingVer] = useState<number | null>(null);
  const [installProgress, setInstallProgress] = useState<string>('');
  const [progressPercent, setProgressPercent] = useState<number>(0);

  const loadEnvironment = async () => {
    try {
      const [s, env] = await Promise.all([GetSettings(), CheckEnvironmentSetup()]);
      setSettings(s);
      setSetupStatus(env);
    } catch (e) {
      console.error('Failed to load settings/environment:', e);
    } finally {
      setLoadingEnv(false);
    }
  };

  useEffect(() => {
    loadEnvironment();

    const cancel = EventsOn('setup:progress', (evt: any) => {
      if (evt?.message) setInstallProgress(evt.message);
      if (typeof evt?.percentage === 'number') setProgressPercent(evt.percentage);
    });

    return () => cancel();
  }, []);

  const handleSave = async () => {
    if (!settings) return;
    await UpdateSettings(settings);
    setSaved(true);
    if (showToast) showToast('Đã lưu cấu hình cài đặt thành công');
    setTimeout(() => setSaved(false), 2000);
  };

  const handleSelectJavaPath = async (path: string) => {
    if (!settings) return;
    const updated = { ...settings, defaultJavaPath: path };
    setSettings(updated);
    await SetDefaultJavaPath(path);
    await loadEnvironment();
    if (showToast) showToast(`Đã chọn Java runtime: ${path}`);
  };

  const handleInstallJava = async (majorVersion: number) => {
    setInstallingVer(majorVersion);
    setInstallProgress(`Đang chuẩn bị tải Java ${majorVersion} (Adoptium Eclipse Temurin)...`);
    setProgressPercent(10);

    try {
      const path = await InstallJavaRuntime(majorVersion);
      await loadEnvironment();
      if (showToast) showToast(`Đã cài đặt thành công Java ${majorVersion} (${path})`);
    } catch (err: any) {
      if (showToast) showToast(`Lỗi cài đặt Java ${majorVersion}: ${err}`);
    } finally {
      setInstallingVer(null);
      setInstallProgress('');
      setProgressPercent(0);
    }
  };

  const handleBrowseJava = async () => {
    try {
      const selected = await BrowseJavaExecutable();
      if (selected) {
        await handleSelectJavaPath(selected);
      }
    } catch (err: any) {
      if (showToast) showToast(`Lỗi duyệt file: ${err}`);
    }
  };

  const isChillTownReady = setupStatus?.hasJava17 || false;

  return (
    <main
      className="flex-1 overflow-y-auto flex flex-col select-none relative z-10"
      style={{ padding: '20px 18px 24px 18px', gap: '22px', boxSizing: 'border-box' }}
    >
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-black text-white flex items-center tracking-tight" style={{ gap: '10px' }}>
            <Settings className="w-6 h-6 text-emerald-400" />
            Cài Đặt & Môi Trường Game
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Quản lý Java runtimes, RAM và cấu hình tự động cho một máy mới hoàn toàn khi chơi Chill Town.
          </p>
        </div>

        <button
          onClick={handleSave}
          className="status-pill rounded-xl bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 font-bold transition-all shadow-md shadow-emerald-500/20 cursor-pointer"
          style={{ height: '40px', padding: '0 20px', gap: '8px', fontSize: '13px' }}
        >
          {saved ? <Check className="w-4 h-4" /> : null}
          <span>{saved ? 'Đã Lưu Thành Công' : 'Lưu Thay Đổi'}</span>
        </button>
      </div>

      {/* CHILL TOWN ENVIRONMENT STATUS BANNER */}
      <div
        className={`glass-panel p-5 rounded-2xl border flex items-center justify-between flex-wrap gap-4 transition-all ${
          loadingEnv
            ? 'border-white/10 bg-black/40'
            : isChillTownReady
            ? 'border-emerald-500/40 bg-emerald-500/[0.06] shadow-lg shadow-emerald-500/10'
            : 'border-amber-500/40 bg-amber-500/[0.08] shadow-lg shadow-amber-500/10'
        }`}
      >
        <div className="flex items-center gap-3.5">
          <div
            className={`w-11 h-11 rounded-2xl flex items-center justify-center shrink-0 border ${
              loadingEnv
                ? 'bg-white/5 border-white/10 text-slate-400'
                : isChillTownReady
                ? 'bg-emerald-500/20 border-emerald-500/30 text-emerald-400'
                : 'bg-amber-500/20 border-amber-500/30 text-amber-400'
            }`}
          >
            {loadingEnv ? (
              <Loader2 className="w-6 h-6 animate-spin text-cyan-400" />
            ) : isChillTownReady ? (
              <CheckCircle2 className="w-6 h-6" />
            ) : (
              <AlertTriangle className="w-6 h-6" />
            )}
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-extrabold text-base text-white">
                Môi Trường Chill Town (Forge 1.20.1)
              </span>
              <span
                className={`status-pill font-mono font-bold text-[10px] uppercase px-2 py-0.5 rounded-md border ${
                  loadingEnv
                    ? 'bg-white/10 text-slate-400 border-white/15'
                    : isChillTownReady
                    ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30'
                    : 'bg-amber-500/20 text-amber-300 border-amber-500/30'
                }`}
              >
                {loadingEnv ? 'ĐANG QUÉT MÔI TRƯỜNG...' : isChillTownReady ? 'ĐÃ SẴN SÀNG' : 'CHƯA CÓ JAVA 17'}
              </span>
            </div>
            <p className="text-xs text-slate-300 mt-0.5">
              {loadingEnv
                ? 'Đang kiểm tra các bản Java runtime và cấu hình phần cứng trên máy...'
                : isChillTownReady
                ? `Đã nhận diện Java 17 chuẩn 64-bit (${setupStatus?.java17Path || 'Adoptium JRE'}). Khởi chạy mượt mà không lỗi crash.`
                : 'Forge 1.20.1 yêu cầu Java 17 64-bit. Máy của bạn chưa có Java 17, hãy bấm nút cài đặt tự động bên dưới.'}
            </p>
          </div>
        </div>

        {!loadingEnv && !isChillTownReady && (
          <button
            onClick={() => handleInstallJava(17)}
            disabled={installingVer !== null}
            className="status-pill px-5 py-2.5 rounded-xl font-bold text-xs bg-gradient-to-r from-amber-500 to-emerald-400 hover:from-amber-400 hover:to-emerald-300 text-slate-950 shadow-md shadow-amber-500/20 cursor-pointer flex items-center gap-2"
          >
            {installingVer === 17 ? (
              <Loader2 className="w-4 h-4 animate-spin text-slate-950" />
            ) : (
              <Sparkles className="w-4 h-4 text-slate-950" />
            )}
            <span>Cài Đặt Java 17 Tự Động (1-Click)</span>
          </button>
        )}
      </div>

      {/* Installing Progress Bar if active */}
      {installingVer !== null && (
        <div className="glass-panel p-4 rounded-xl border border-cyan-500/30 bg-cyan-950/20 flex flex-col gap-2">
          <div className="flex items-center justify-between text-xs font-semibold text-cyan-300">
            <span className="flex items-center gap-2">
              <Loader2 className="w-4 h-4 animate-spin" />
              <span>{installProgress}</span>
            </span>
            <span className="font-mono">{Math.round(progressPercent)}%</span>
          </div>
          <div className="w-full h-2 rounded-full bg-white/10 overflow-hidden">
            <div
              className="h-full bg-gradient-to-r from-cyan-400 to-emerald-400 transition-all duration-200"
              style={{ width: `${Math.min(100, Math.max(5, progressPercent))}%` }}
            />
          </div>
        </div>
      )}

      {/* JAVA RUNTIMES MANAGER SECTION */}
      <div className="glass-panel p-5 space-y-4">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <div>
            <h2 className="text-sm font-bold text-white flex items-center gap-2 uppercase tracking-wider text-slate-300">
              <Coffee className="w-4 h-4 text-amber-400" />
              <span>Quản Lý Môi Trường Java (Java Runtimes)</span>
            </h2>
            <p className="text-[11px] text-slate-400 mt-0.5">
              Bạn có thể chọn bản Java mặc định sử dụng hoặc tải các bản Java độc lập (Java 8, 17, 21) không cần quyền Admin.
            </p>
          </div>

          <button
            onClick={handleBrowseJava}
            className="status-pill px-3.5 py-1.5 rounded-xl bg-white/[0.04] hover:bg-white/[0.08] border border-white/10 text-slate-300 hover:text-white text-xs font-semibold flex items-center gap-1.5 cursor-pointer"
          >
            <FileSearch className="w-3.5 h-3.5 text-cyan-400" />
            <span>Duyệt javaw.exe thủ công</span>
          </button>
        </div>

        {/* List of Detected / Downloaded Java Runtimes */}
        <div className="space-y-2 pt-1">
          {loadingEnv ? (
            <div className="p-4 rounded-2xl bg-black/30 border border-white/5 flex items-center gap-3 text-xs text-slate-400">
              <Loader2 className="w-4 h-4 text-cyan-400 animate-spin" />
              <span>Đang quét các bản Java trên ổ cứng...</span>
            </div>
          ) : setupStatus?.allRuntimes && setupStatus.allRuntimes.length > 0 ? (
            setupStatus.allRuntimes.map((rt, idx) => {
              const isSelected =
                (settings?.defaultJavaPath && settings.defaultJavaPath.toLowerCase() === rt.path.toLowerCase()) ||
                (!settings?.defaultJavaPath && rt.path === setupStatus.selectedJava);

              return (
                <div
                  key={idx}
                  onClick={() => handleSelectJavaPath(rt.path)}
                  className={`p-3.5 rounded-2xl border transition-all flex items-center justify-between text-xs cursor-pointer ${
                    isSelected
                      ? 'border-emerald-500/50 bg-emerald-500/[0.08] shadow-md shadow-emerald-500/10'
                      : 'border-white/5 hover:border-white/15 bg-black/30'
                  }`}
                >
                  <div className="space-y-1 min-w-0 pr-3 flex-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-extrabold text-white text-sm">Java {rt.majorVersion}</span>
                      <span className="text-[10.5px] text-slate-400 font-mono">({rt.version})</span>

                      {rt.majorVersion === 17 && (
                        <span className="status-pill font-mono font-bold text-[9.5px] px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-300 border border-emerald-500/25">
                          ★ Khuyên Dùng Cho Chill Town
                        </span>
                      )}

                      {rt.majorVersion === 21 && (
                        <span className="status-pill font-mono font-bold text-[9.5px] px-2 py-0.5 rounded bg-blue-500/15 text-blue-300 border border-blue-500/25">
                          Minecraft 1.20.5+
                        </span>
                      )}

                      {rt.majorVersion === 8 && (
                        <span className="status-pill font-mono font-bold text-[9.5px] px-2 py-0.5 rounded bg-purple-500/15 text-purple-300 border border-purple-500/25">
                          Bản Cũ (1.12 - 1.16)
                        </span>
                      )}

                      {isSelected && (
                        <span className="status-pill font-mono font-bold text-[9.5px] px-2 py-0.5 rounded bg-emerald-500/25 text-emerald-300 border border-emerald-500/40">
                          ✓ Đang Chọn
                        </span>
                      )}
                    </div>
                    <div className="font-mono text-[11px] text-slate-400 truncate">{rt.path}</div>
                  </div>

                  <div className="flex items-center gap-3 shrink-0">
                    <span className="text-[11px] font-mono text-slate-500 font-semibold">
                      {rt.is64Bit ? '64-Bit' : '32-Bit'}
                    </span>
                    <button
                      type="button"
                      className={`status-pill px-3 py-1 rounded-xl font-bold text-xs ${
                        isSelected
                          ? 'bg-emerald-500 text-slate-950 font-bold'
                          : 'bg-white/5 hover:bg-white/10 text-slate-300'
                      }`}
                    >
                      {isSelected ? 'Đang Dùng' : 'Chọn'}
                    </button>
                  </div>
                </div>
              );
            })
          ) : (
            <p className="text-xs text-slate-400 p-2">
              Chưa phát hiện bản Java độc lập nào. Hệ thống đang dùng Java đi kèm của Windows.
            </p>
          )}
        </div>

        {/* 1-Click Java Download Cards for Other Versions */}
        <div className="pt-3 border-t border-white/5 flex flex-col gap-2.5">
          <div className="text-xs font-bold text-slate-300 uppercase tracking-wider">
            Tải Thêm Java Bản Quyền Khuyên Dùng (Eclipse Temurin JRE):
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
            {/* Java 17 */}
            <button
              type="button"
              onClick={() => handleInstallJava(17)}
              disabled={installingVer !== null}
              className={`p-3 rounded-xl border text-left transition-all flex items-center justify-between cursor-pointer ${
                setupStatus?.hasJava17
                  ? 'border-emerald-500/20 bg-emerald-500/[0.04] text-slate-300'
                  : 'border-white/10 hover:border-emerald-500/40 bg-white/[0.02] hover:bg-white/[0.06] text-white'
              }`}
            >
              <div className="flex flex-col">
                <span className="font-bold text-xs flex items-center gap-1.5">
                  <span>Java 17 (LTS)</span>
                  {setupStatus?.hasJava17 && <Check className="w-3.5 h-3.5 text-emerald-400" />}
                </span>
                <span className="text-[10px] text-slate-400">Cho Chill Town & 1.20.1</span>
              </div>
              <Download className="w-4 h-4 text-emerald-400 shrink-0" />
            </button>

            {/* Java 21 */}
            <button
              type="button"
              onClick={() => handleInstallJava(21)}
              disabled={installingVer !== null}
              className={`p-3 rounded-xl border text-left transition-all flex items-center justify-between cursor-pointer ${
                setupStatus?.hasJava21
                  ? 'border-blue-500/20 bg-blue-500/[0.04] text-slate-300'
                  : 'border-white/10 hover:border-blue-500/40 bg-white/[0.02] hover:bg-white/[0.06] text-white'
              }`}
            >
              <div className="flex flex-col">
                <span className="font-bold text-xs flex items-center gap-1.5">
                  <span>Java 21 (LTS)</span>
                  {setupStatus?.hasJava21 && <Check className="w-3.5 h-3.5 text-blue-400" />}
                </span>
                <span className="text-[10px] text-slate-400">Cho Minecraft 1.21+</span>
              </div>
              <Download className="w-4 h-4 text-blue-400 shrink-0" />
            </button>

            {/* Java 8 */}
            <button
              type="button"
              onClick={() => handleInstallJava(8)}
              disabled={installingVer !== null}
              className={`p-3 rounded-xl border text-left transition-all flex items-center justify-between cursor-pointer ${
                setupStatus?.hasJava8
                  ? 'border-purple-500/20 bg-purple-500/[0.04] text-slate-300'
                  : 'border-white/10 hover:border-purple-500/40 bg-white/[0.02] hover:bg-white/[0.06] text-white'
              }`}
            >
              <div className="flex flex-col">
                <span className="font-bold text-xs flex items-center gap-1.5">
                  <span>Java 8 (Legacy)</span>
                  {setupStatus?.hasJava8 && <Check className="w-3.5 h-3.5 text-purple-400" />}
                </span>
                <span className="text-[10px] text-slate-400">Cho bản cũ (1.12 - 1.16.5)</span>
              </div>
              <Download className="w-4 h-4 text-purple-400 shrink-0" />
            </button>
          </div>
        </div>
      </div>

      {/* Hardware Specs Panel */}
      <div className="glass-panel p-5 space-y-4">
        <h2 className="text-sm font-bold text-white flex items-center gap-2 uppercase tracking-wider text-slate-400">
          <Cpu className="w-4 h-4 text-cyan-400" />
          Phần Cứng Phát Hiện Được
        </h2>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-xs">
          <div className="bg-black/30 p-3.5 rounded-xl border border-white/5 space-y-1">
            <div className="text-slate-400">Bộ Vi Xử Lý (CPU)</div>
            <div className="font-bold text-white truncate">{stats?.cpuModel || 'Đang quét...'}</div>
            <div className="text-[11px] text-emerald-400 font-semibold">{stats?.cpuCores || 0} Cores Logic</div>
          </div>

          <div className="bg-black/30 p-3.5 rounded-xl border border-white/5 space-y-1">
            <div className="text-slate-400">Bộ Nhớ RAM Tổng</div>
            <div className="font-bold text-white">
              {stats ? Math.round(stats.totalRamMb / 1024) : 0} GB RAM Vật Lý
            </div>
            <div className="text-[11px] text-cyan-400 font-semibold">
              Còn trống: {stats ? Math.round(stats.freeRamMb / 1024) : 0} GB
            </div>
          </div>

          <div className="bg-black/30 p-3.5 rounded-xl border border-white/5 space-y-1">
            <div className="text-slate-400">Card Đồ Họa (GPU)</div>
            <div className="font-bold text-white truncate">{stats?.gpuModel || 'Card tích hợp'}</div>
            <div className="text-[11px] text-slate-400">Hỗ trợ Shaders Iris / Sodium</div>
          </div>
        </div>
      </div>

      {/* Memory & Settings Configuration */}
      {settings && (
        <div className="glass-panel p-5 space-y-5">
          <h2 className="text-sm font-bold text-white flex items-center gap-2 uppercase tracking-wider text-slate-400">
            <HardDrive className="w-4 h-4 text-emerald-400" />
            Bộ Nhớ Mặc Định Cho Game Mới
          </h2>

          <div className="space-y-4 text-xs">
            <div className="space-y-2">
              <div className="flex justify-between text-slate-300 font-semibold">
                <span>RAM tối đa mặc định (Max RAM):</span>
                <span className="text-emerald-400 font-bold">{settings.defaultMaxRam} MB ({Math.round(settings.defaultMaxRam / 1024)}GB)</span>
              </div>
              <input
                type="range"
                min="1024"
                max="16384"
                step="512"
                value={settings.defaultMaxRam}
                onChange={(e) =>
                  setSettings({ ...settings, defaultMaxRam: Number(e.target.value) })
                }
                className="w-full accent-emerald-500 cursor-pointer"
              />
            </div>

            {/* Launcher Directory */}
            <div className="space-y-1.5 pt-2">
              <label className="text-slate-300 font-semibold">Thư mục dữ liệu Launcher:</label>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  readOnly
                  value={settings.launcherDir}
                  className="flex-1 px-3 py-2 rounded-xl bg-black/40 border border-white/10 text-slate-300 font-mono text-xs focus:outline-none"
                />
                <button
                  onClick={() => OpenFolder(settings.launcherDir)}
                  className="btn-secondary px-4 py-2 text-xs font-semibold cursor-pointer shrink-0"
                >
                  <FolderOpen className="w-3.5 h-3.5 text-cyan-400" />
                  Mở Thư Mục
                </button>
              </div>
            </div>

            {/* Auto close toggle */}
            <div className="flex items-center justify-between pt-2">
              <div>
                <div className="font-semibold text-white">Tự động ẩn / đóng Launcher khi vào game</div>
                <div className="text-[11px] text-slate-400">
                  Giúp giải phóng 100% tài nguyên CPU/RAM cho máy tính khi Minecraft đang chạy.
                </div>
              </div>
              <input
                type="checkbox"
                checked={settings.autoCloseOnLaunch}
                onChange={(e) =>
                  setSettings({ ...settings, autoCloseOnLaunch: e.target.checked })
                }
                className="w-5 h-5 accent-emerald-500 rounded cursor-pointer"
              />
            </div>
          </div>
        </div>
      )}

      {/* Auto-Update Launcher Section */}
      <div className="glass-panel p-5 space-y-4 border border-cyan-500/30 bg-gradient-to-r from-cyan-950/20 via-slate-900/60 to-emerald-950/20">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Zap className="w-5 h-5 text-cyan-400" />
            <h2 className="text-sm font-bold text-white uppercase tracking-wider">
              Tự Động Cập Nhật CamLauncher (In-Place Hot-Swap)
            </h2>
          </div>
          <span className="text-xs px-2 py-0.5 rounded-full bg-cyan-500/20 text-cyan-300 font-mono font-bold border border-cyan-500/30">
            Phiên bản hiện tại: v2.4.1
          </span>
        </div>

        <p className="text-xs text-slate-300 leading-relaxed">
          Cơ chế cập nhật độc quyền trên Windows: Tự động tải bản nhị phân mới và hoán đổi trực tiếp trong bộ nhớ. Người dùng <strong>không cần phải tải lại file cài đặt hay ấn wizard setup thủ công</strong>.
        </p>

        <div className="flex items-center justify-between pt-2 border-t border-white/5">
          <div className="text-xs text-slate-400">
            {launcherUpdate?.hasUpdate ? (
              <span className="text-cyan-300 font-bold">
                ✨ Phát hiện bản cập nhật mới v{launcherUpdate.latestVersion} ({launcherUpdate.title || 'Mới'})
              </span>
            ) : (
              <span>CamLauncher đang ở phiên bản mới nhất.</span>
            )}
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={onCheckLauncherUpdate}
              className="btn-secondary px-4 py-2 text-xs font-semibold cursor-pointer"
            >
              Kiểm Tra Cập Nhật
            </button>
            {launcherUpdate?.hasUpdate && (
              <button
                onClick={onOpenLauncherUpdate}
                className="btn-primary px-4 py-2 text-xs font-bold cursor-pointer shadow-lg shadow-cyan-500/20"
              >
                Cập Nhật Ngay
              </button>
            )}
          </div>
        </div>
      </div>

      {/* About Box */}
      <div className="glass-panel p-5 space-y-2 border border-emerald-500/20 bg-gradient-to-r from-emerald-950/20 via-slate-900/40 to-slate-900/60">
        <div className="flex items-center gap-2 text-white font-bold text-sm">
          <Shield className="w-4 h-4 text-emerald-400" />
          CamLauncher Alpha - Owlys Apps
        </div>
        <p className="text-xs text-slate-300 leading-relaxed">
          Được thiết kế trên nền tảng <strong>Go + Windows Native WebView2</strong>. Khởi chạy tức thì trong 50ms, tiêu thụ ~35MB RAM, cam kết không quảng cáo, không theo dõi và tương thích tuyệt đối mọi phiên bản Minecraft.
        </p>
      </div>
    </main>
  );
};
