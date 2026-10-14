import React, { useState, useEffect } from 'react';
import {
  X,
  Sliders,
  Package,
  Sparkles,
  Layers,
  FolderOpen,
  Trash2,
  Search,
  RotateCw,
  Check,
  Cpu,
  Zap,
  AlertTriangle,
  FileCode,
  Image,
  ChevronDown,
} from 'lucide-react';
import { models } from '../../wailsjs/go/models';
import {
  UpdateInstance,
  GetInstanceFiles,
  ToggleInstanceFile,
  DeleteInstanceFile,
  OpenInstanceSubFolder,
  OpenInstanceFolder,
  OpenFolder,
  BrowseJavaExecutable,
} from '../../wailsjs/go/main/App';

interface InstanceSettingsModalProps {
  instance: models.Instance | null;
  stats: models.SystemStats | null;
  isOpen: boolean;
  onClose: () => void;
  onUpdated: (inst: models.Instance) => void;
  showToast: (msg: string) => void;
}

type TabType = 'general' | 'mods' | 'resourcepacks' | 'shaders';

export const InstanceSettingsModal: React.FC<InstanceSettingsModalProps> = ({
  instance,
  stats,
  isOpen,
  onClose,
  onUpdated,
  showToast,
}) => {
  if (!isOpen || !instance) return null;

  const [activeTab, setActiveTab] = useState<TabType>('general');

  // General tab states
  const [name, setName] = useState(instance.name);
  const [allocatedRam, setAllocatedRam] = useState(instance.maxMemoryMb || 4096);
  const [jvmArgs, setJvmArgs] = useState(instance.jvmArgs || '');
  const [javaPath, setJavaPath] = useState(instance.javaPath || '');
  const [showAdvanced, setShowAdvanced] = useState(
    Boolean(instance.javaPath || (instance.jvmArgs && instance.jvmArgs.trim().length > 0))
  );
  const [savingGeneral, setSavingGeneral] = useState(false);

  // Files state (mods, resourcepacks, shaders)
  const [fileItems, setFileItems] = useState<models.InstanceFileItem[]>([]);
  const [loadingFiles, setLoadingFiles] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'enabled' | 'disabled'>('all');

  const totalSystemRamMb = stats?.totalRamMb || 16384;
  const totalSystemRamGb = Math.round(totalSystemRamMb / 1024);

  // Reset general form when instance changes
  useEffect(() => {
    if (instance) {
      setName(instance.name);
      setAllocatedRam(instance.maxMemoryMb || 4096);
      setJvmArgs(instance.jvmArgs || '');
      setJavaPath(instance.javaPath || '');
      if (instance.javaPath || (instance.jvmArgs && instance.jvmArgs.trim().length > 0)) {
        setShowAdvanced(true);
      }
    }
  }, [instance]);

  // Load files when switching to file tabs
  const loadFiles = async (type: TabType) => {
    if (!instance || type === 'general') return;
    setLoadingFiles(true);
    try {
      const folderKey =
        type === 'mods'
          ? 'mods'
          : type === 'resourcepacks'
          ? 'resourcepacks'
          : 'shaderpacks';
      const items = await GetInstanceFiles(instance.id, folderKey);
      setFileItems(items || []);
    } catch (err: any) {
      console.error('Failed to load files:', err);
      showToast(`Lỗi quét file: ${err}`);
    } finally {
      setLoadingFiles(false);
    }
  };

  useEffect(() => {
    if (activeTab !== 'general') {
      setSearchQuery('');
      loadFiles(activeTab);
    }
  }, [activeTab, instance]);

  // Handle Save General Settings
  const handleSaveGeneral = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!instance) return;

    setSavingGeneral(true);
    try {
      const updated = models.Instance.createFrom({
        ...instance,
        name: name.trim() || instance.name,
        minMemoryMb: allocatedRam,
        maxMemoryMb: allocatedRam,
        jvmArgs: jvmArgs.trim(),
        javaPath: javaPath.trim(),
      });

      const saved = await UpdateInstance(updated);
      const finalInst = saved || updated;
      onUpdated(finalInst);
      showToast(`Đã lưu cấu hình và đồng bộ thư mục: "${finalInst.name}"`);
    } catch (err: any) {
      showToast(`Lỗi lưu cài đặt: ${err}`);
    } finally {
      setSavingGeneral(false);
    }
  };

  // Toggle file enable/disable
  const handleToggleFile = async (item: models.InstanceFileItem) => {
    if (!instance) return;
    try {
      await ToggleInstanceFile(instance.id, item.fileType, item.fileName, !item.enabled);
      // Optimistically update
      setFileItems((prev) =>
        prev.map((f) =>
          f.fileName === item.fileName
            ? {
                ...f,
                enabled: !item.enabled,
                fileName: !item.enabled
                  ? f.fileName.replace(/\.disabled$/, '')
                  : f.fileName.endsWith('.disabled')
                  ? f.fileName
                  : f.fileName + '.disabled',
              }
            : f
        )
      );
      showToast(`${!item.enabled ? 'Đã bật' : 'Đã tắt'}: ${item.name}`);
    } catch (err: any) {
      showToast(`Không thể đổi trạng thái file: ${err}`);
      loadFiles(activeTab);
    }
  };

  // Delete file
  const handleDeleteFile = async (item: models.InstanceFileItem) => {
    if (!instance) return;
    if (!confirm(`Bạn có chắc muốn xóa vĩnh viễn file "${item.fileName}"?`)) return;

    try {
      await DeleteInstanceFile(instance.id, item.fileType, item.fileName);
      setFileItems((prev) => prev.filter((f) => f.fileName !== item.fileName));
      showToast(`Đã xóa: ${item.name}`);
    } catch (err: any) {
      showToast(`Lỗi khi xóa file: ${err}`);
    }
  };

  // Open Subfolder
  const handleOpenFolder = async (folderType: string) => {
    if (!instance) return;
    try {
      await OpenInstanceSubFolder(instance.id, folderType);
      showToast(`Đã mở thư mục ${folderType}`);
    } catch (err: any) {
      showToast(`Lỗi mở thư mục: ${err}`);
    }
  };

  // Open Root Instance Folder in File Explorer
  const handleOpenRootFolder = async () => {
    if (!instance) return;
    try {
      if (instance.path) {
        await OpenFolder(instance.path);
      } else {
        await OpenInstanceFolder(instance.id);
      }
      showToast('Đã mở thư mục Instance trong Windows File Explorer');
    } catch (err: any) {
      showToast(`Lỗi mở thư mục: ${err}`);
    }
  };

  // Browse Java Executable
  const handleBrowseJava = async () => {
    try {
      const selected = await BrowseJavaExecutable();
      if (selected && selected.trim()) {
        setJavaPath(selected.trim());
        showToast(`Đã chọn Java: ${selected.trim()}`);
      }
    } catch (err: any) {
      showToast(`Không thể chọn Java: ${err}`);
    }
  };

  // Preset Aikar Flags
  const applyAikarFlags = () => {
    setJvmArgs(
      '-XX:+UseG1GC -XX:+ParallelRefProcEnabled -XX:MaxGCPauseMillis=200 -XX:+UnlockExperimentalVMOptions -XX:+DisableExplicitGC -XX:+AlwaysPreTouch -XX:G1NewSizePercent=30 -XX:G1MaxNewSizePercent=40 -XX:G1ReservePercent=20 -XX:G1HeapWastePercent=5 -XX:G1MixedGCCountTarget=4 -XX:InitiatingHeapOccupancyPercent=15 -XX:G1MixedGCLiveThresholdPercent=90 -XX:G1RSetUpdatingPauseTimePercent=5 -XX:SurvivorRatio=32 -XX:+PerfDisableSharedMem -XX:Provider.SunPKCS11.disableIncompatibilityCheck=true'
    );
    showToast('Đã áp dụng bộ cờ Aikar Flags tối ưu FPS!');
  };

  // Filtered files
  const filteredFiles = fileItems.filter((item) => {
    const matchesSearch =
      item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.fileName.toLowerCase().includes(searchQuery.toLowerCase());
    if (!matchesSearch) return false;
    if (filterStatus === 'enabled') return item.enabled;
    if (filterStatus === 'disabled') return !item.enabled;
    return true;
  });

  const enabledCount = fileItems.filter((f) => f.enabled).length;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-md p-4 select-none">
      <div
        className="glass-panel modal-enter w-full flex flex-col rounded-3xl border border-white/15 shadow-2xl relative overflow-hidden"
        style={{
          maxWidth: '920px',
          height: '660px',
          maxHeight: '92vh',
          backgroundColor: 'rgba(12, 16, 26, 0.96)',
        }}
      >
        {/* Header */}
        <div
          className="flex items-center justify-between border-b border-white/10 shrink-0"
          style={{ padding: '16px 24px' }}
        >
          <div className="flex items-center" style={{ gap: '14px' }}>
            <div className="w-10 h-10 rounded-xl bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center text-emerald-400 shrink-0">
              <Sliders className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center" style={{ gap: '8px' }}>
                <h2 className="text-lg font-black text-white tracking-tight">{instance.name}</h2>
                <span
                  className="status-pill rounded-md bg-emerald-500/15 text-emerald-300 border border-emerald-500/25 font-mono"
                  style={{ height: '22px', padding: '0 8px', fontSize: '10.5px' }}
                >
                  MC {instance.gameVersion} • {instance.loader?.toUpperCase() || 'VANILLA'}
                </span>
              </div>
              {/* Interactive Folder Path Button */}
              <div className="flex items-center mt-1">
                <button
                  type="button"
                  onClick={handleOpenRootFolder}
                  className="group flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-white/[0.04] hover:bg-emerald-500/15 border border-white/10 hover:border-emerald-500/30 text-slate-400 hover:text-emerald-300 transition-all cursor-pointer text-left max-w-lg"
                  title="Bấm để mở thư mục Instance này trong Windows File Explorer"
                >
                  <FolderOpen className="w-3.5 h-3.5 text-slate-400 group-hover:text-emerald-400 shrink-0 transition-colors" />
                  <span className="text-[11px] font-mono truncate max-w-sm">{instance.path}</span>
                  <span className="text-[10px] font-sans font-semibold text-emerald-400/90 group-hover:text-emerald-300 shrink-0 ml-1">
                    • Mở thư mục
                  </span>
                </button>
              </div>
            </div>
          </div>

          <button
            onClick={onClose}
            className="w-8 h-8 rounded-xl hover:bg-white/10 flex items-center justify-center text-slate-400 hover:text-white transition-colors cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tab Navigation Bar */}
        <div
          className="flex items-center justify-between border-b border-white/10 bg-black/20 shrink-0"
          style={{ padding: '0 24px' }}
        >
          <div className="flex items-center" style={{ gap: '4px' }}>
            {[
              { id: 'general' as TabType, label: 'Cài Đặt Chung & RAM', icon: Sliders },
              { id: 'mods' as TabType, label: 'Quản Lý Mods', icon: Package },
              { id: 'resourcepacks' as TabType, label: 'Resource Packs', icon: Image },
              { id: 'shaders' as TabType, label: 'Shader Packs', icon: Sparkles },
            ].map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`status-pill cursor-pointer font-bold text-xs transition-all relative ${
                    isActive ? 'text-emerald-300' : 'text-slate-400 hover:text-slate-200'
                  }`}
                  style={{ height: '46px', padding: '0 16px', gap: '8px' }}
                >
                  <Icon className={`w-4 h-4 ${isActive ? 'text-emerald-400' : 'text-slate-400'}`} />
                  <span>{tab.label}</span>
                  {tab.id !== 'general' && activeTab === tab.id && (
                    <span
                      className="status-pill rounded-full bg-emerald-500/20 text-emerald-300 font-mono"
                      style={{ height: '18px', padding: '0 6px', fontSize: '10px' }}
                    >
                      {fileItems.length}
                    </span>
                  )}
                  {isActive && (
                    <div className="absolute bottom-0 left-2 right-2 h-0.5 bg-gradient-to-r from-emerald-400 to-teal-400 rounded-full" />
                  )}
                </button>
              );
            })}
          </div>

          {activeTab !== 'general' && (
            <button
              onClick={() =>
                handleOpenFolder(
                  activeTab === 'mods'
                    ? 'mods'
                    : activeTab === 'resourcepacks'
                    ? 'resourcepacks'
                    : 'shaderpacks'
                )
              }
              className="status-pill text-xs font-semibold text-slate-300 hover:text-emerald-300 hover:bg-white/[0.06] rounded-xl transition-all border border-white/10"
              style={{ height: '32px', padding: '0 12px', gap: '6px' }}
              title="Mở thư mục trong Windows File Explorer"
            >
              <FolderOpen className="w-3.5 h-3.5 text-emerald-400" />
              <span>Mở Thư Mục</span>
            </button>
          )}
        </div>

        {/* Modal Body */}
        <div className="flex-1 overflow-y-auto" style={{ padding: '24px' }}>
          {/* TAB 1: GENERAL & RAM SETTINGS */}
          {activeTab === 'general' && (
            <form onSubmit={handleSaveGeneral} className="flex flex-col" style={{ gap: '22px' }}>
              {/* Instance Name */}
              <div className="flex flex-col" style={{ gap: '8px' }}>
                <label className="text-xs font-bold text-slate-300 uppercase tracking-wider">
                  Tên Bản Cài Đặt (Instance Name)
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-4 py-2.5 rounded-xl bg-black/40 border border-white/10 text-white font-semibold text-sm focus:outline-none focus:border-emerald-500/50"
                />
                <span className="text-[11px] text-slate-400">
                  Thư mục instance trên ổ đĩa sẽ tự động cập nhật và đồng bộ theo tên này.
                </span>
              </div>

              {/* RAM Allocation Section - Single Slider UX */}
              <div
                className="glass-panel p-5 rounded-2xl border border-white/10 flex flex-col"
                style={{ gap: '16px', background: 'rgba(15, 23, 42, 0.65)' }}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center" style={{ gap: '8px' }}>
                    <Zap className="w-4 h-4 text-emerald-400" />
                    <span className="text-sm font-bold text-white uppercase tracking-wider">
                      Bộ Nhớ RAM Cấp Phát
                    </span>
                  </div>
                  <div className="flex items-center" style={{ gap: '8px' }}>
                    <span className="text-xs font-mono text-slate-400">
                      Tổng RAM máy: <strong className="text-slate-200">{totalSystemRamGb} GB</strong>
                    </span>
                    <span className="text-xs px-2.5 py-0.5 rounded-md bg-emerald-500/20 text-emerald-300 font-mono font-bold border border-emerald-500/30">
                      {Math.round(allocatedRam / 1024)} GB ({allocatedRam} MB)
                    </span>
                  </div>
                </div>

                {/* Single Range Slider */}
                <div className="flex flex-col" style={{ gap: '8px' }}>
                  <input
                    type="range"
                    min={1024}
                    max={totalSystemRamMb}
                    step={512}
                    value={allocatedRam}
                    onChange={(e) => setAllocatedRam(Number(e.target.value))}
                    className="w-full accent-emerald-400 cursor-pointer h-2 bg-black/50 rounded-lg"
                  />
                  <div className="flex justify-between text-[11px] font-mono text-slate-500">
                    <span>1 GB (Tối thiểu)</span>
                    <span>{Math.round(totalSystemRamGb / 2)} GB</span>
                    <span>{totalSystemRamGb} GB (Toàn bộ RAM)</span>
                  </div>
                </div>

                {/* Quick Presets */}
                <div className="flex items-center flex-wrap" style={{ gap: '8px', paddingTop: '2px' }}>
                  <span className="text-xs text-slate-400 mr-1 font-semibold">Gợi ý nhanh:</span>
                  {[
                    { label: '2 GB', value: 2048 },
                    { label: '4 GB (Khuyên dùng)', value: 4096 },
                    { label: '6 GB', value: 6144 },
                    { label: '8 GB', value: 8192 },
                    ...(totalSystemRamGb >= 16 ? [{ label: '12 GB', value: 12288 }] : []),
                  ].map((preset) => {
                    const isSelected = allocatedRam === preset.value;
                    return (
                      <button
                        key={preset.value}
                        type="button"
                        onClick={() => setAllocatedRam(preset.value)}
                        className={`status-pill rounded-lg text-xs transition-all cursor-pointer font-semibold ${
                          isSelected
                            ? 'bg-emerald-500/25 text-emerald-300 border border-emerald-500/50 shadow-sm shadow-emerald-500/10'
                            : 'bg-white/[0.04] text-slate-400 hover:text-white hover:bg-white/[0.08] border border-white/5'
                        }`}
                        style={{ height: '30px', padding: '0 12px' }}
                      >
                        {preset.label}
                      </button>
                    );
                  })}
                </div>

                {allocatedRam > totalSystemRamMb * 0.85 && (
                  <div
                    className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-300 text-xs flex items-center"
                    style={{ gap: '8px' }}
                  >
                    <AlertTriangle className="w-4 h-4 shrink-0 text-amber-400" />
                    <span>
                      Cảnh báo: Bạn đang cấp phát hơn 85% RAM tổng của máy tính. Điều này có thể khiến Windows bị giật lag nếu chạy đa nhiệm!
                    </span>
                  </div>
                )}
              </div>

              {/* Advanced Settings Accordion (Java Path & JVM Flags) */}
              <div className="border border-white/10 rounded-2xl overflow-hidden bg-black/20 transition-all">
                <button
                  type="button"
                  onClick={() => setShowAdvanced(!showAdvanced)}
                  className="w-full px-5 py-3.5 flex items-center justify-between text-left hover:bg-white/[0.04] transition-colors cursor-pointer"
                >
                  <div className="flex items-center" style={{ gap: '10px' }}>
                    <div className="w-6 h-6 rounded-lg bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center text-cyan-400">
                      <Cpu className="w-3.5 h-3.5" />
                    </div>
                    <div className="flex items-center" style={{ gap: '8px' }}>
                      <span className="text-xs font-bold text-slate-200 uppercase tracking-wider">
                        Cài Đặt Nâng Cao (JVM & Java Path)
                      </span>
                      {(javaPath.trim() || jvmArgs.trim()) && (
                        <span className="status-pill text-[10px] px-2 py-0.5 rounded-md bg-cyan-500/15 text-cyan-300 font-mono font-semibold border border-cyan-500/30">
                          Đang tùy chỉnh
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center text-slate-400 text-xs font-medium" style={{ gap: '6px' }}>
                    <span>{showAdvanced ? 'Thu gọn' : 'Mở rộng'}</span>
                    <ChevronDown
                      className={`w-4 h-4 transition-transform duration-200 ${
                        showAdvanced ? 'rotate-180 text-emerald-400' : ''
                      }`}
                    />
                  </div>
                </button>

                {showAdvanced && (
                  <div
                    className="p-5 border-t border-white/10 flex flex-col modal-enter"
                    style={{ gap: '18px', background: 'rgba(10, 15, 25, 0.45)' }}
                  >
                    {/* 1. Java Runtime Selector & File Browser */}
                    <div className="flex flex-col" style={{ gap: '8px' }}>
                      <div className="flex items-center justify-between">
                        <label className="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center" style={{ gap: '6px' }}>
                          <Cpu className="w-4 h-4 text-emerald-400" />
                          <span>Đường Dẫn Java (Java Runtime)</span>
                        </label>
                        {javaPath && (
                          <button
                            type="button"
                            onClick={() => {
                              setJavaPath('');
                              showToast('Đã chuyển về Java mặc định tự động');
                            }}
                            className="text-xs text-slate-400 hover:text-white transition-colors underline cursor-pointer"
                          >
                            Đặt lại tự động
                          </button>
                        )}
                      </div>

                      <div className="flex items-center" style={{ gap: '8px' }}>
                        <select
                          value={
                            javaPath === ''
                              ? ''
                              : (stats?.javaRuntimes || []).some((r) => r.path === javaPath)
                              ? javaPath
                              : '__custom__'
                          }
                          onChange={(e) => {
                            if (e.target.value === '__custom__') {
                              if (!javaPath) setJavaPath(' ');
                            } else {
                              setJavaPath(e.target.value);
                            }
                          }}
                          className="flex-1 px-3.5 py-2.5 rounded-xl bg-black/40 border border-white/10 text-white text-xs font-medium focus:outline-none focus:border-emerald-500/50 cursor-pointer"
                        >
                          <option value="" className="bg-slate-900 text-slate-200">
                            [Mặc định] Tự động (Đi kèm Launcher / Tự tải - Khuyên dùng)
                          </option>
                          {(stats?.javaRuntimes || []).map((rt, idx) => (
                            <option key={idx} value={rt.path} className="bg-slate-900 text-slate-200">
                              Java {rt.majorVersion} ({rt.version || 'Đã cài'}) - {rt.path}
                            </option>
                          ))}
                          <option value="__custom__" className="bg-slate-900 text-cyan-300">
                            Tùy chọn đường dẫn khác...
                          </option>
                        </select>

                        <button
                          type="button"
                          onClick={handleBrowseJava}
                          className="status-pill rounded-xl bg-white/[0.06] hover:bg-emerald-500/20 text-slate-200 hover:text-emerald-300 border border-white/10 hover:border-emerald-500/30 text-xs font-semibold shrink-0 cursor-pointer transition-all"
                          style={{ height: '38px', padding: '0 14px', gap: '6px' }}
                          title="Mở hộp thoại Windows để tìm file javaw.exe"
                        >
                          <FolderOpen className="w-4 h-4 text-emerald-400" />
                          <span>Duyệt...</span>
                        </button>
                      </div>

                      {/* Custom path input when custom or non-empty */}
                      {javaPath.trim() !== '' && (
                        <div className="flex items-center" style={{ gap: '8px' }}>
                          <input
                            type="text"
                            value={javaPath}
                            onChange={(e) => setJavaPath(e.target.value)}
                            placeholder="C:\Program Files\Java\jdk-21\bin\javaw.exe"
                            className="w-full px-3.5 py-2 rounded-xl bg-black/50 border border-white/10 text-emerald-300 font-mono text-xs focus:outline-none focus:border-emerald-500/50"
                          />
                        </div>
                      )}
                      <span className="text-[11px] text-slate-500">
                        Launcher sẽ tự động sử dụng Java 8 / 17 / 21 tương thích chuẩn với bản cài đặt nếu để mặc định.
                      </span>
                    </div>

                    {/* 2. JVM Flags */}
                    <div className="flex flex-col" style={{ gap: '8px' }}>
                      <div className="flex items-center justify-between">
                        <label className="text-xs font-bold text-slate-300 uppercase tracking-wider flex items-center" style={{ gap: '6px' }}>
                          <FileCode className="w-4 h-4 text-cyan-400" />
                          <span>Tham Số Khởi Động Java (JVM Flags)</span>
                        </label>
                        <div className="flex items-center" style={{ gap: '10px' }}>
                          <button
                            type="button"
                            onClick={applyAikarFlags}
                            className="text-xs text-emerald-400 hover:text-emerald-300 font-semibold cursor-pointer underline"
                          >
                            Áp dụng Aikar Flags (Tối ưu FPS)
                          </button>
                          {jvmArgs && (
                            <button
                              type="button"
                              onClick={() => {
                                setJvmArgs('');
                                showToast('Đã xóa JVM Flags');
                              }}
                              className="text-xs text-slate-400 hover:text-rose-400 transition-colors cursor-pointer"
                            >
                              Xóa cờ
                            </button>
                          )}
                        </div>
                      </div>
                      <textarea
                        rows={3}
                        value={jvmArgs}
                        onChange={(e) => setJvmArgs(e.target.value)}
                        placeholder="Ví dụ: -XX:+UseG1GC -XX:+ParallelRefProcEnabled -XX:MaxGCPauseMillis=200..."
                        className="w-full px-3.5 py-2.5 rounded-xl bg-black/40 border border-white/10 text-white font-mono text-xs focus:outline-none focus:border-emerald-500/50 resize-none leading-relaxed"
                      />
                      <span className="text-[11px] text-slate-500">
                        Chỉ thêm tham số phụ (không cần -Xmx và -Xms vì đã được cài đặt tự động qua thanh trượt RAM ở trên).
                      </span>
                    </div>
                  </div>
                )}
              </div>

              {/* Footer Save & Cancel Buttons */}
              <div className="flex items-center justify-end pt-2" style={{ gap: '12px' }}>
                <button
                  type="button"
                  onClick={onClose}
                  className="status-pill rounded-xl bg-white/[0.04] hover:bg-white/[0.08] text-slate-400 hover:text-white border border-white/10 transition-all font-semibold cursor-pointer"
                  style={{ height: '42px', padding: '0 20px', fontSize: '13px' }}
                >
                  Hủy
                </button>
                <button
                  type="submit"
                  disabled={savingGeneral}
                  className="status-pill rounded-xl bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 font-bold transition-all shadow-md shadow-emerald-500/20 cursor-pointer disabled:opacity-50"
                  style={{ height: '42px', padding: '0 24px', gap: '8px', fontSize: '13px' }}
                >
                  <Check className="w-4 h-4" />
                  <span>{savingGeneral ? 'Đang Lưu...' : 'Lưu Thay Đổi'}</span>
                </button>
              </div>
            </form>
          )}

          {/* TAB 2, 3, 4: FILE MANAGERS (MODS / RESOURCEPACKS / SHADERS) */}
          {activeTab !== 'general' && (
            <div className="flex flex-col h-full" style={{ gap: '16px' }}>
              {/* Search & Status Filters */}
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
                <div className="relative flex-1 max-w-md">
                  <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder={`Tìm kiếm ${activeTab === 'mods' ? 'mod' : activeTab === 'resourcepacks' ? 'resource pack' : 'shader'}...`}
                    className="w-full pl-9 pr-3.5 py-2 rounded-xl bg-black/40 border border-white/10 text-white placeholder-slate-500 text-xs focus:outline-none focus:border-emerald-500/50 font-medium"
                  />
                </div>

                <div className="flex items-center" style={{ gap: '8px' }}>
                  <div className="flex items-center bg-black/40 rounded-xl p-1 border border-white/10 text-xs">
                    {(['all', 'enabled', 'disabled'] as const).map((st) => (
                      <button
                        key={st}
                        type="button"
                        onClick={() => setFilterStatus(st)}
                        className={`status-pill rounded-lg font-semibold transition cursor-pointer ${
                          filterStatus === st
                            ? 'bg-emerald-500/20 text-emerald-300 font-bold'
                            : 'text-slate-400 hover:text-white'
                        }`}
                        style={{ height: '28px', padding: '0 10px', fontSize: '11px' }}
                      >
                        {st === 'all'
                          ? `Tất cả (${fileItems.length})`
                          : st === 'enabled'
                          ? `Bật (${enabledCount})`
                          : `Tắt (${fileItems.length - enabledCount})`}
                      </button>
                    ))}
                  </div>

                  <button
                    onClick={() => loadFiles(activeTab)}
                    className="w-8 h-8 rounded-xl bg-white/[0.04] hover:bg-white/[0.08] border border-white/10 flex items-center justify-center text-slate-400 hover:text-white transition cursor-pointer"
                    title="Làm mới danh sách"
                  >
                    <RotateCw className={`w-3.5 h-3.5 ${loadingFiles ? 'animate-spin text-emerald-400' : ''}`} />
                  </button>
                </div>
              </div>

              {/* Files List Table */}
              {loadingFiles ? (
                <div className="py-24 flex flex-col items-center justify-center" style={{ gap: '12px' }}>
                  <div className="w-8 h-8 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin" />
                  <p className="text-xs text-slate-400">Đang quét thư mục {activeTab}...</p>
                </div>
              ) : filteredFiles.length === 0 ? (
                <div
                  className="glass-panel flex flex-col items-center justify-center py-20 rounded-2xl border border-dashed border-white/10 text-center"
                  style={{ gap: '12px', background: 'rgba(15, 23, 42, 0.4)' }}
                >
                  <Package className="w-10 h-10 text-slate-500 stroke-1" />
                  <div className="flex flex-col" style={{ gap: '4px' }}>
                    <p className="text-sm font-bold text-slate-300">
                      {searchQuery
                        ? 'Không tìm thấy file nào khớp từ khóa'
                        : `Chưa có file nào trong thư mục ${activeTab}`}
                    </p>
                    <p className="text-xs text-slate-500 max-w-sm">
                      {activeTab === 'mods'
                        ? 'Bạn có thể tải mod từ tab "Kho Mods" hoặc bấm "Mở Thư Mục" để dán file .jar vào.'
                        : 'Bấm nút "Mở Thư Mục" ở trên để sao chép file .zip của bạn vào đây.'}
                    </p>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col" style={{ gap: '8px' }}>
                  {filteredFiles.map((file) => (
                    <div
                      key={file.fileName}
                      className={`glass-panel flex items-center justify-between rounded-xl border transition-all ${
                        file.enabled
                          ? 'border-white/[0.08] hover:border-white/20 bg-slate-900/60'
                          : 'border-white/[0.04] bg-black/40 opacity-70'
                      }`}
                      style={{ padding: '12px 16px' }}
                    >
                      {/* Left: Info */}
                      <div className="flex items-center min-w-0 flex-1" style={{ gap: '12px' }}>
                        <div
                          className={`w-2.5 h-2.5 rounded-full shrink-0 ${
                            file.enabled ? 'bg-emerald-400 shadow-sm shadow-emerald-400/50' : 'bg-slate-600'
                          }`}
                        />
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center" style={{ gap: '8px' }}>
                            <span
                              className={`text-xs font-bold truncate ${
                                file.enabled ? 'text-white' : 'text-slate-400 line-through'
                              }`}
                            >
                              {file.name}
                            </span>
                            <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-white/5 text-slate-400 shrink-0">
                              {file.fileName.endsWith('.disabled') ? '.disabled' : activeTab === 'mods' ? '.jar' : '.zip'}
                            </span>
                          </div>
                          <div
                            className="text-[11px] text-slate-500 font-mono mt-0.5 flex items-center"
                            style={{ gap: '8px' }}
                          >
                            <span>{(file.sizeBytes / (1024 * 1024)).toFixed(2)} MB</span>
                            <span>•</span>
                            <span>{file.modTime}</span>
                          </div>
                        </div>
                      </div>

                      {/* Right: Actions */}
                      <div className="flex items-center shrink-0 ml-3" style={{ gap: '12px' }}>
                        {/* Toggle Switch */}
                        <label
                          className="relative inline-flex items-center cursor-pointer"
                          title={file.enabled ? 'Bấm để tắt' : 'Bấm để bật'}
                        >
                          <input
                            type="checkbox"
                            checked={file.enabled}
                            onChange={() => handleToggleFile(file)}
                            className="sr-only peer"
                          />
                          <div className="w-11 h-6 bg-slate-800 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-500 border border-white/10" />
                        </label>

                        {/* Delete Button */}
                        <button
                          type="button"
                          onClick={() => handleDeleteFile(file)}
                          className="w-8 h-8 rounded-lg hover:bg-rose-500/20 text-slate-500 hover:text-rose-400 flex items-center justify-center transition cursor-pointer"
                          title="Xóa vĩnh viễn"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
