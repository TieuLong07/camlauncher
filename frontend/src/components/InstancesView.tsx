import { Dialog } from './Dialog';
import React, { useState, useEffect } from 'react';
import {
  Ellipsis,
  Plus,
  Layers,
  FolderOpen,
  Trash2,
  Check,
  X,
  RotateCw,
  Settings,
  Sparkles,
  Zap,
  Shield,
  Box,
  FlaskConical,
  Swords,
  Pickaxe,
  Skull,
  Loader2,
  Wand2,
  Cpu,
  Search,
} from 'lucide-react';
import { models } from '../../wailsjs/go/models';
import {
  GetVersionManifest,
  GetLoaderVersions,
  CreateInstance,
  DeleteInstance,
  OpenInstanceFolder,
  OpenFolder,
  InstallFPSBoostPack,
} from '../../wailsjs/go/main/App';

interface InstancesViewProps {
  instances: models.Instance[];
  activeInstance: models.Instance | null;
  stats?: models.SystemStats | null;
  onSelectInstance: (inst: models.Instance) => void;
  onOpenSettings: (inst: models.Instance) => void;
  onRefresh: () => Promise<unknown>;
  showToast?: (msg: string) => void;
}

type LoaderType = 'vanilla' | 'fabric' | 'forge' | 'neoforge';

export const InstancesView: React.FC<InstancesViewProps> = ({
  instances,
  activeInstance,
  stats,
  onSelectInstance,
  onOpenSettings,
  onRefresh,
  showToast,
}) => {
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [manifest, setManifest] = useState<models.VersionManifest | null>(null);

  // Form states for creating an instance
  const [name, setName] = useState('');
  const [isNameCustomized, setIsNameCustomized] = useState(false);
  const [selectedVersion, setSelectedVersion] = useState('1.21.1');
  const [versionFilter, setVersionFilter] = useState<'releases' | 'snapshots' | 'all'>('releases');
  const [versionSearch, setVersionSearch] = useState('');

  const [loader, setLoader] = useState<LoaderType>('fabric');
  const [loaderVersions, setLoaderVersions] = useState<string[]>([]);
  const [selectedLoaderVersion, setSelectedLoaderVersion] = useState<string>('');
  const [recommendedLoaderVersion, setRecommendedLoaderVersion] = useState<string>('');
  const [autoRecommended, setAutoRecommended] = useState(true);
  const [loadingLoaders, setLoadingLoaders] = useState(false);

  const [selectedIcon, setSelectedIcon] = useState('cube');
  const [maxRAM, setMaxRAM] = useState(4096);
  const [installFPSBoost, setInstallFPSBoost] = useState(true);

  const [creating, setCreating] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const totalSystemRAM = stats?.totalRamMb || 14336;

  // Load Mojang version manifest on mount
  useEffect(() => {
    GetVersionManifest()
      .then((m) => {
        setManifest(m);
        if (m?.latest?.release) {
          setSelectedVersion(m.latest.release);
        }
      })
      .catch((err) => console.error('GetVersionManifest error:', err));
  }, []);

  // Fetch available loader versions whenever selectedVersion or loader changes
  useEffect(() => {
    if (!showCreateModal) return;

    if (loader === 'vanilla') {
      setLoaderVersions([]);
      setSelectedLoaderVersion('');
      setRecommendedLoaderVersion('');
      return;
    }

    let isSubscribed = true;
    setLoadingLoaders(true);

    GetLoaderVersions(selectedVersion, loader)
      .then((res: models.LoaderVersionResult) => {
        if (!isSubscribed) return;
        const vers = res?.versions || [];
        setLoaderVersions(vers);
        const rec = res?.recommended || (vers.length > 0 ? vers[0] : '');
        setRecommendedLoaderVersion(rec);

        if (autoRecommended || !selectedLoaderVersion || !vers.includes(selectedLoaderVersion)) {
          setSelectedLoaderVersion(rec);
        }
      })
      .catch((err) => {
        console.warn('GetLoaderVersions error:', err);
        if (isSubscribed) {
          setLoaderVersions([]);
          setSelectedLoaderVersion('');
        }
      })
      .finally(() => {
        if (isSubscribed) setLoadingLoaders(false);
      });

    return () => {
      isSubscribed = false;
    };
  }, [selectedVersion, loader, showCreateModal]);

  // Update default name when version or loader changes, if user hasn't typed custom name
  useEffect(() => {
    if (!isNameCustomized) {
      const loaderPart =
        loader === 'vanilla'
          ? 'Vanilla'
          : loader === 'fabric'
          ? 'Fabric'
          : loader === 'forge'
          ? 'Forge'
          : 'NeoForge';
      setName(`${selectedVersion} ${loaderPart}`);
    }
  }, [selectedVersion, loader, isNameCustomized]);

  // Filter versions according to release/snapshot and search term
  const filteredVersions = (manifest?.versions || []).filter((v) => {
    if (versionFilter === 'releases' && v.type !== 'release') return false;
    if (versionFilter === 'snapshots' && v.type !== 'snapshot') return false;
    if (versionSearch.trim() && !v.id.toLowerCase().includes(versionSearch.toLowerCase().trim())) {
      return false;
    }
    return true;
  });

  const handleManualRefresh = async () => {
    setIsRefreshing(true);
    try {
      await onRefresh();
      showToast?.('Đã làm mới danh sách bản cài.');
    } catch (error) {
      showToast?.(`Không thể quét bản cài. ${error}`);
    } finally {
      setIsRefreshing(false);
    }
  };

  const handleResetAutoName = () => {
    setIsNameCustomized(false);
    const loaderPart =
      loader === 'vanilla'
        ? 'Vanilla'
        : loader === 'fabric'
        ? 'Fabric'
        : loader === 'forge'
        ? 'Forge'
        : 'NeoForge';
    setName(`${selectedVersion} ${loaderPart}`);
  };

  const handleCreate = async () => {
    if (!name.trim()) return;
    setCreating(true);

    try {
      const instName = name.trim();
      const loaderVal = loader === 'vanilla' ? '' : loader;
      const loaderVerVal = loader === 'vanilla' ? '' : selectedLoaderVersion;

      const newInst = await CreateInstance(
        instName,
        selectedVersion,
        loaderVal,
        loaderVerVal,
        selectedIcon,
        1024,
        maxRAM
      );

      // Auto-install FPS boost pack if requested (mainly for Fabric)
      if (installFPSBoost && loader === 'fabric' && newInst?.id) {
        if (showToast) showToast('Đang cài đặt gói tối ưu Cam FPS Boost...');
        try {
          await InstallFPSBoostPack(newInst.id);
        } catch (boostErr) {
          console.warn('FPS boost install non-fatal error:', boostErr);
        }
      }

      setShowCreateModal(false);
      setName('');
      setIsNameCustomized(false);
      await onRefresh();
      onSelectInstance(newInst);
      if (showToast) showToast(`Đã tạo thành công instance "${instName}"`);
    } catch (err: any) {
      if (showToast) showToast(`Lỗi tạo instance: ${err}`);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string, instName: string) => {
    if (confirm(`Bạn có chắc muốn xóa bản cài "${instName}"? (Saves và mods sẽ bị xóa)`)) {
      try {
        await DeleteInstance(id);
        await onRefresh();
        if (showToast) showToast(`Đã xóa instance "${instName}"`);
      } catch (err: any) {
        if (showToast) showToast(`Không thể xóa: ${err}`);
      }
    }
  };

  const iconOptions = [
    { id: 'cube', label: 'Khối Minecraft', icon: <Box className="w-4 h-4 text-orange-400" /> },
    { id: 'flask', label: 'Bình thuốc', icon: <FlaskConical className="w-4 h-4 text-purple-400" /> },
    { id: 'swords', label: 'Gươm chiến', icon: <Swords className="w-4 h-4 text-rose-400" /> },
    { id: 'pickaxe', label: 'Cúp đào', icon: <Pickaxe className="w-4 h-4 text-amber-400" /> },
    { id: 'skull', label: 'Boss / Hardcore', icon: <Skull className="w-4 h-4 text-purple-300" /> },
    { id: 'zap', label: 'Tia sét FPS', icon: <Zap className="w-4 h-4 text-yellow-400" /> },
    { id: 'shield', label: 'Khiên bảo vệ', icon: <Shield className="w-4 h-4 text-blue-400" /> },
    { id: 'sparkles', label: 'Ma thuật', icon: <Sparkles className="w-4 h-4 text-orange-400" /> },
  ];

  return (
    <main
      className="flex-1 overflow-y-auto flex flex-col select-none relative z-10"
      style={{ padding: '22px 24px 28px 24px', gap: '22px', boxSizing: 'border-box' }}
    >
      <div className="instance-header">
        <div><h1>Bản cài cá nhân</h1><p>{instances.length} bản cài trên máy</p></div>
        <div className="instance-header-actions">
          <button className="cam-button" onClick={handleManualRefresh} disabled={isRefreshing}>
            <RotateCw size={16} className={isRefreshing ? 'animate-spin' : ''} />
            {isRefreshing ? 'Đang quét…' : 'Quét lại máy'}
          </button>
          <button className="cam-button cam-primary" onClick={() => {
            setShowCreateModal(true);
            setIsNameCustomized(false);
          }}><Plus size={17} /> Thêm bản cài</button>
        </div>
      </div>

      {instances.length === 0 ? (
        <p className="instance-empty">Chưa có bản cài trên máy. Thêm một bản để bắt đầu.</p>
      ) : (
        <ul className="instance-list" aria-label="Bản cài trên máy">
          {instances.map((instance) => {
            const selected = activeInstance?.id === instance.id;
            return (
              <li key={instance.id} className="instance-row" data-selected={selected}>
                <button className="instance-select" aria-pressed={selected}
                  onClick={() => onSelectInstance(instance)}>
                  <span className="readiness-icon"><Layers size={19} /></span>
                  <span className="instance-name">
                    <strong>{instance.name}</strong>
                    <small>Minecraft {instance.gameVersion} · {instance.loader || 'Vanilla'}{instance.loaderVersion ? ` ${instance.loaderVersion}` : ''}</small>
                  </span>
                  <span className="instance-ram">{Math.round(instance.maxMemoryMb / 1024)} GB RAM</span>
                  <span className="instance-select-label">{selected && <Check size={15} />}{selected ? 'Đang chọn' : 'Chọn'}</span>
                </button>
                <details className="instance-menu">
                  <summary className="cam-button cam-icon" aria-label={`Thao tác với ${instance.name}`}><Ellipsis size={19} /></summary>
                  <div className="instance-menu-actions">
                    <button className="cam-button" onClick={() => OpenInstanceFolder(instance.id).catch((error) => showToast?.(String(error)))}>
                      <FolderOpen size={16} /> Mở thư mục
                    </button>
                    <button className="cam-button" onClick={() => onOpenSettings(instance)}><Settings size={16} /> Cấu hình</button>
                    {instances.length > 1 && <button className="cam-button cam-danger" onClick={() => handleDelete(instance.id, instance.name)}>
                      <Trash2 size={16} /> Xóa bản cài
                    </button>}
                  </div>
                </details>
              </li>
            );
          })}
        </ul>
      )}
      <button className="cam-button self-start" onClick={() => OpenFolder('').catch((error) => showToast?.(String(error)))}>
        <FolderOpen size={16} /> Mở thư mục launcher
      </button>

      {showCreateModal && (
        <Dialog titleId="create-instance-title" busy={creating} onClose={() => setShowCreateModal(false)}>
          <div
            className="create-instance-panel glass-panel relative flex flex-col"
            style={{ padding: '22px 26px', gap: '14px' }}
          >
            {/* Modal Header */}
            <div className="flex shrink-0 items-center justify-between border-b border-white/10 pb-3">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-xl bg-orange-500/15 border border-orange-500/30 flex items-center justify-center text-orange-400">
                  <Plus className="w-5 h-5" />
                </div>
                <div>
                  <h2 id="create-instance-title" className="text-xl font-extrabold text-white tracking-tight">
                    Tạo bản cài
                  </h2>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Chọn phiên bản Minecraft và trình tải mod.
                  </p>
                </div>
              </div>
              <button
                disabled={creating}
                onClick={() => setShowCreateModal(false)}
                aria-label="Đóng hộp thoại"
                className="w-8 h-8 rounded-xl hover:bg-white/10 flex items-center justify-center text-slate-400 hover:text-white transition cursor-pointer"
              >
                <X className="w-4 h-4" aria-hidden="true" />
              </button>
            </div>

            {/* Modal Body */}
            <div className="flex flex-col gap-4 text-xs min-h-0 overflow-y-auto pr-2">
              {/* SECTION 1: MOD LOADER SELECTOR (4 CARDS) */}
              <div className="flex flex-col gap-2">
                <label className="text-slate-200 font-bold uppercase tracking-wider text-[11px] flex items-center justify-between">
                  <span>1. Chọn Loại Modloader</span>
                  <span className="text-[10px] text-slate-400 font-normal">
                    {loader === 'vanilla'
                      ? 'Nguyên bản không mod'
                      : loader === 'fabric'
                      ? 'Tối ưu FPS cao'
                      : loader === 'forge'
                      ? 'Nhiều modpack lớn'
                      : 'Forge thế hệ mới'}
                  </span>
                </label>

                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                  {/* Vanilla */}
                  <button
                    type="button"
                    onClick={() => setLoader('vanilla')}
                    className={`p-3 rounded-2xl border text-left transition-all flex flex-col justify-between gap-2 cursor-pointer ${
                      loader === 'vanilla'
                        ? 'border-emerald-500/50 bg-orange-500/15 shadow-md shadow-emerald-500/10'
                        : 'border-white/10 bg-white/[0.02] hover:bg-white/[0.05] text-slate-400'
                    }`}
                  >
                    <div className="flex items-center justify-between w-full">
                      <Box className={`w-5 h-5 ${loader === 'vanilla' ? 'text-orange-400' : 'text-slate-400'}`} />
                      {loader === 'vanilla' && <Check className="w-3.5 h-3.5 text-orange-400" />}
                    </div>
                    <div>
                      <div className="font-bold text-white text-xs">Vanilla</div>
                      <div className="text-[10px] text-slate-400 mt-0.5">Gốc, siêu mượt</div>
                    </div>
                  </button>

                  {/* Fabric */}
                  <button
                    type="button"
                    onClick={() => {
                      setLoader('fabric');
                      setInstallFPSBoost(true);
                    }}
                    className={`p-3 rounded-2xl border text-left transition-all flex flex-col justify-between gap-2 cursor-pointer ${
                      loader === 'fabric'
                        ? 'border-orange-500/50 bg-cyan-500/15 shadow-md shadow-cyan-500/10'
                        : 'border-white/10 bg-white/[0.02] hover:bg-white/[0.05] text-slate-400'
                    }`}
                  >
                    <div className="flex items-center justify-between w-full">
                      <Zap className={`w-5 h-5 ${loader === 'fabric' ? 'text-orange-400' : 'text-slate-400'}`} />
                      {loader === 'fabric' && <Check className="w-3.5 h-3.5 text-orange-400" />}
                    </div>
                    <div>
                      <div className="font-bold text-white text-xs">Fabric</div>
                      <div className="text-[10px] text-orange-300/80 mt-0.5">Tối ưu FPS cao</div>
                    </div>
                  </button>

                  {/* Forge */}
                  <button
                    type="button"
                    onClick={() => {
                      setLoader('forge');
                      setInstallFPSBoost(false);
                    }}
                    className={`p-3 rounded-2xl border text-left transition-all flex flex-col justify-between gap-2 cursor-pointer ${
                      loader === 'forge'
                        ? 'border-purple-500/50 bg-purple-500/15 shadow-md shadow-purple-500/10'
                        : 'border-white/10 bg-white/[0.02] hover:bg-white/[0.05] text-slate-400'
                    }`}
                  >
                    <div className="flex items-center justify-between w-full">
                      <FlaskConical className={`w-5 h-5 ${loader === 'forge' ? 'text-purple-400' : 'text-slate-400'}`} />
                      {loader === 'forge' && <Check className="w-3.5 h-3.5 text-purple-400" />}
                    </div>
                    <div>
                      <div className="font-bold text-white text-xs">Forge</div>
                      <div className="text-[10px] text-purple-300/80 mt-0.5">Modpack lớn</div>
                    </div>
                  </button>

                  {/* NeoForge */}
                  <button
                    type="button"
                    onClick={() => {
                      setLoader('neoforge');
                      setInstallFPSBoost(false);
                    }}
                    className={`p-3 rounded-2xl border text-left transition-all flex flex-col justify-between gap-2 cursor-pointer ${
                      loader === 'neoforge'
                        ? 'border-amber-500/50 bg-amber-500/15 shadow-md shadow-amber-500/10'
                        : 'border-white/10 bg-white/[0.02] hover:bg-white/[0.05] text-slate-400'
                    }`}
                  >
                    <div className="flex items-center justify-between w-full">
                      <Sparkles className={`w-5 h-5 ${loader === 'neoforge' ? 'text-amber-400' : 'text-slate-400'}`} />
                      {loader === 'neoforge' && <Check className="w-3.5 h-3.5 text-amber-400" />}
                    </div>
                    <div>
                      <div className="font-bold text-white text-xs">NeoForge</div>
                      <div className="text-[10px] text-amber-300/80 mt-0.5">1.20.2+ Hiện đại</div>
                    </div>
                  </button>
                </div>
              </div>

              {/* SECTION 2: MINECRAFT VERSION PICKER (WITH QUICK CHIPS & SEARCH) */}
              <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <label className="text-slate-200 font-bold uppercase tracking-wider text-[11px]">
                    2. Chọn Phiên Bản Minecraft
                  </label>

                  {/* Version type switch */}
                  <div className="flex items-center bg-black/40 border border-white/10 rounded-xl p-0.5">
                    <button
                      type="button"
                      onClick={() => setVersionFilter('releases')}
                      className={`px-2.5 py-1 rounded-lg text-[10.5px] font-bold transition cursor-pointer ${
                        versionFilter === 'releases'
                          ? 'bg-orange-500/20 text-orange-300 border border-orange-500/30'
                          : 'text-slate-400 hover:text-white'
                      }`}
                    >
                      Chính Thức
                    </button>
                    <button
                      type="button"
                      onClick={() => setVersionFilter('snapshots')}
                      className={`px-2.5 py-1 rounded-lg text-[10.5px] font-bold transition cursor-pointer ${
                        versionFilter === 'snapshots'
                          ? 'bg-orange-500/20 text-orange-300 border border-orange-500/30'
                          : 'text-slate-400 hover:text-white'
                      }`}
                    >
                      Thử Nghiệm
                    </button>
                    <button
                      type="button"
                      onClick={() => setVersionFilter('all')}
                      className={`px-2.5 py-1 rounded-lg text-[10.5px] font-bold transition cursor-pointer ${
                        versionFilter === 'all'
                          ? 'bg-orange-500/20 text-orange-300 border border-orange-500/30'
                          : 'text-slate-400 hover:text-white'
                      }`}
                    >
                      Tất Cả
                    </button>
                  </div>
                </div>

                {/* Quick Version Chips */}
                <div className="flex flex-wrap gap-1.5 pt-0.5">
                  {['1.21.1', '1.20.4', '1.20.1', '1.19.4', '1.18.2', '1.16.5', '1.12.2', '1.8.9'].map((v) => (
                    <button
                      key={v}
                      type="button"
                      onClick={() => setSelectedVersion(v)}
                      className={`status-pill px-2.5 py-1 rounded-lg font-mono text-[11px] transition cursor-pointer ${
                        selectedVersion === v
                          ? 'bg-orange-400 text-slate-950 font-extrabold shadow-sm'
                          : 'bg-white/[0.04] hover:bg-white/[0.08] text-slate-300 border border-white/10'
                      }`}
                    >
                      {v}
                    </button>
                  ))}
                </div>

                {/* Search & Dropdown row */}
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                  <div className="relative flex items-center sm:col-span-1">
                    <Search className="w-3.5 h-3.5 text-slate-400 absolute left-3 pointer-events-none" />
                    <input
                      type="text"
                      placeholder="Lọc phiên bản..."
                      value={versionSearch}
                      onChange={(e) => setVersionSearch(e.target.value)}
                      className="w-full pl-8 pr-3 py-2.5 rounded-xl bg-black/40 border border-white/10 text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500/50 text-xs font-mono"
                    />
                  </div>

                  <select
                    value={selectedVersion}
                    onChange={(e) => setSelectedVersion(e.target.value)}
                    className="sm:col-span-2 px-3.5 py-2.5 rounded-xl bg-slate-900 border border-white/10 text-white focus:outline-none focus:border-emerald-500/50 text-xs font-mono font-bold"
                  >
                    {filteredVersions.length > 0 ? (
                      filteredVersions.map((v) => (
                        <option key={v.id} value={v.id}>
                          Minecraft {v.id} {v.type === 'release' ? '(Chính thức)' : `(${v.type})`}
                        </option>
                      ))
                    ) : (
                      <option value={selectedVersion}>Minecraft {selectedVersion}</option>
                    )}
                  </select>
                </div>
              </div>

              {/* SECTION 3: MODLOADER VERSION SELECTOR (DYNAMIC) */}
              {loader !== 'vanilla' && (
                <div className="flex flex-col gap-2 p-3.5 rounded-2xl bg-white/[0.02] border border-white/10">
                  <div className="flex items-center justify-between">
                    <label className="text-slate-200 font-bold uppercase tracking-wider text-[11px] flex items-center gap-2">
                      <span>3. Phiên Bản {loader.toUpperCase()} Loader</span>
                      {loadingLoaders && <Loader2 className="w-3 h-3 text-orange-400 animate-spin" />}
                    </label>

                    <label className="flex items-center gap-1.5 text-[11px] text-slate-300 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={autoRecommended}
                        onChange={(e) => {
                          setAutoRecommended(e.target.checked);
                          if (e.target.checked && recommendedLoaderVersion) {
                            setSelectedLoaderVersion(recommendedLoaderVersion);
                          }
                        }}
                        className="rounded border-white/20 accent-emerald-500"
                      />
                      <span>Tự động chọn bản khuyên dùng</span>
                    </label>
                  </div>

                  <div className="flex items-center gap-2">
                    <select
                      value={selectedLoaderVersion}
                      disabled={loadingLoaders || autoRecommended}
                      onChange={(e) => setSelectedLoaderVersion(e.target.value)}
                      className="flex-1 px-3.5 py-2.5 rounded-xl bg-slate-900 border border-white/10 text-white focus:outline-none focus:border-emerald-500/50 text-xs font-mono font-bold disabled:opacity-60"
                    >
                      {loaderVersions.length > 0 ? (
                        loaderVersions.map((lv) => (
                          <option key={lv} value={lv}>
                            {loader.toUpperCase()} {lv} {lv === recommendedLoaderVersion ? '★ (Khuyên dùng - Recommended)' : ''}
                          </option>
                        ))
                      ) : (
                        <option value="">
                          {loadingLoaders ? 'Đang truy vấn danh sách...' : 'Mặc định / Tự động'}
                        </option>
                      )}
                    </select>

                    {recommendedLoaderVersion && (
                      <span className="status-pill px-2.5 py-1.5 rounded-xl font-mono text-[10.5px] font-bold bg-orange-500/15 text-orange-300 border border-orange-500/30 shrink-0">
                        Khuyên dùng: {recommendedLoaderVersion}
                      </span>
                    )}
                  </div>
                </div>
              )}

              {/* SECTION 4: INSTANCE NAME & ICON */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div className="sm:col-span-2 flex flex-col gap-1.5">
                  <div className="flex items-center justify-between">
                    <label className="text-slate-200 font-bold uppercase tracking-wider text-[11px]">
                      4. Tên Bản Cài Đặt
                    </label>
                    <button
                      type="button"
                      onClick={handleResetAutoName}
                      className="text-[10px] text-orange-400 hover:text-orange-300 flex items-center gap-1 cursor-pointer"
                      title="Đặt lại tên tự động"
                    >
                      <Wand2 className="w-3 h-3" />
                      <span>Tự Động Đặt Tên</span>
                    </button>
                  </div>
                  <input
                    type="text"
                    placeholder="Ví dụ: My Survival 1.21, PvP Ranked..."
                    value={name}
                    onChange={(e) => {
                      setName(e.target.value);
                      setIsNameCustomized(true);
                    }}
                    className="w-full px-3.5 py-2.5 rounded-xl bg-black/40 border border-white/10 text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500/50 text-xs font-bold"
                  />
                </div>

                {/* Icon Selection */}
                <div className="flex flex-col gap-1.5">
                  <label className="text-slate-200 font-bold uppercase tracking-wider text-[11px]">
                    Biểu Tượng
                  </label>
                  <div className="flex items-center gap-1 overflow-x-auto p-1 bg-black/40 border border-white/10 rounded-xl">
                    {iconOptions.slice(0, 5).map((ico) => (
                      <button
                        key={ico.id}
                        type="button"
                        onClick={() => setSelectedIcon(ico.id)}
                        className={`p-2 rounded-lg transition cursor-pointer ${
                          selectedIcon === ico.id
                            ? 'bg-emerald-500/25 border border-emerald-500/50 shadow-sm'
                            : 'hover:bg-white/10'
                        }`}
                        title={ico.label}
                      >
                        {ico.icon}
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              {/* SECTION 5: RAM ALLOCATION */}
              <div className="flex flex-col gap-2 p-3.5 rounded-2xl bg-white/[0.02] border border-white/10">
                <div className="flex justify-between items-center text-slate-300">
                  <span className="font-bold text-[11px] uppercase tracking-wider text-slate-200 flex items-center gap-1.5">
                    <Cpu className="w-3.5 h-3.5 text-orange-400" />
                    <span>5. Phân Bổ Bộ Nhớ RAM</span>
                  </span>
                  <span className="font-mono text-orange-400 font-extrabold text-sm">
                    {maxRAM} MB ({Math.round(maxRAM / 1024)} GB)
                  </span>
                </div>

                <input
                  type="range"
                  min="1024"
                  max={Math.min(16384, Math.max(8192, totalSystemRAM))}
                  step="512"
                  value={maxRAM}
                  onChange={(e) => setMaxRAM(Number(e.target.value))}
                  className="w-full accent-emerald-500 cursor-pointer"
                />

                <div className="flex items-center justify-between pt-1">
                  <div className="flex items-center gap-1.5">
                    {[2048, 4096, 6144, 8192, 10240].map((ram) => (
                      <button
                        key={ram}
                        type="button"
                        onClick={() => setMaxRAM(ram)}
                        className={`status-pill px-2.5 py-1 rounded-lg font-mono text-[10.5px] transition cursor-pointer ${
                          maxRAM === ram
                            ? 'bg-orange-400 text-slate-950 font-bold'
                            : 'bg-white/5 hover:bg-white/10 text-slate-400'
                        }`}
                      >
                        {ram / 1024} GB
                      </button>
                    ))}
                  </div>

                  <span className="text-[10px] text-slate-400">
                    Hệ thống: {Math.round(totalSystemRAM / 1024)} GB RAM
                  </span>
                </div>
              </div>

              {/* SECTION 6: FPS BOOST TOGGLE (FABRIC ONLY) */}
              {loader === 'fabric' && (
                <div className="flex items-center justify-between p-3 rounded-2xl bg-emerald-500/[0.06] border border-emerald-500/25">
                  <div className="flex items-center gap-2.5">
                    <div className="w-7 h-7 rounded-lg bg-orange-500/20 border border-orange-500/30 flex items-center justify-center text-orange-400 shrink-0">
                      <Zap className="w-4 h-4" />
                    </div>
                    <div className="flex flex-col">
                      <span className="font-bold text-white text-xs">
                        Tự động cài Cam FPS Boost Pack
                      </span>
                      <span className="text-[10.5px] text-slate-400">
                        Bao gồm Sodium, Lithium, Iris Shaders chống giật lag
                      </span>
                    </div>
                  </div>

                  <input
                    type="checkbox"
                    checked={installFPSBoost}
                    onChange={(e) => setInstallFPSBoost(e.target.checked)}
                    className="w-4 h-4 rounded border-white/20 accent-emerald-500 cursor-pointer"
                  />
                </div>
              )}
            </div>

            {/* Modal Actions */}
            <div className="flex shrink-0 items-center justify-end pt-3 border-t border-white/10 gap-3">
              <button
                type="button"
                disabled={creating}
                onClick={() => setShowCreateModal(false)}
                className="status-pill px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white transition cursor-pointer"
                style={{ height: '38px', padding: '0 18px' }}
              >
                Hủy
              </button>
              <button
                type="button"
                onClick={handleCreate}
                disabled={creating || !name.trim()}
                className="status-pill px-6 py-2 text-xs font-extrabold bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 rounded-xl transition-all shadow-md shadow-emerald-500/20 cursor-pointer disabled:opacity-50 flex items-center gap-2"
                style={{ height: '38px', padding: '0 22px' }}
              >
                {creating ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin text-slate-950" />
                    <span>Đang Khởi Tạo...</span>
                  </>
                ) : (
                  <>
                    <Plus className="w-4 h-4 text-slate-950 stroke-[3]" />
                    <span>Tạo Bản Cài Đặt</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </Dialog>
      )}
    </main>
  );
};
