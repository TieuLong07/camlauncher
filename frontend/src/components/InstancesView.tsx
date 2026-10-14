import React, { useState, useEffect } from 'react';
import {
  Plus,
  Layers,
  FolderOpen,
  Trash2,
  HardDrive,
  Check,
  X,
  RotateCw,
  ExternalLink,
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
  onRefresh: () => void;
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

  const handleManualRefresh = () => {
    setIsRefreshing(true);
    onRefresh();
    if (showToast) showToast('Đã quét lại danh sách bản cài đặt cục bộ');
    setTimeout(() => setIsRefreshing(false), 800);
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
      onRefresh();
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
        onRefresh();
        if (showToast) showToast(`Đã xóa instance "${instName}"`);
      } catch (err: any) {
        if (showToast) showToast(`Không thể xóa: ${err}`);
      }
    }
  };

  const getInstanceBadge = (inst: models.Instance) => {
    const n = inst.name.toLowerCase();
    if (n.includes('boosted') || n.includes('fps')) {
      return {
        label: 'TỐI ƯU FPS',
        className: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/25',
      };
    }
    if (n.includes('cooking') || n.includes('valenhardt') || inst.loader === 'forge') {
      return {
        label: 'MODPACK LỚN',
        className: 'bg-purple-500/15 text-purple-300 border-purple-500/25',
      };
    }
    if (inst.loader === 'vanilla' || !inst.loader) {
      return {
        label: 'GỐC',
        className: 'bg-slate-500/15 text-slate-300 border-slate-500/25',
      };
    }
    return {
      label: inst.loader.toUpperCase(),
      className: 'bg-blue-500/15 text-blue-300 border-blue-500/25',
    };
  };

  const getInstanceDesc = (inst: models.Instance) => {
    const n = inst.name.toLowerCase();
    if (n.includes('cooking')) {
      return "Hơn 246 mod Farmer's Delight, Tech & Cooking bản quyền";
    }
    if (n.includes('boosted')) {
      return 'Nạp sẵn Sodium, Lithium, Iris Shaders tối ưu 300% FPS';
    }
    if (inst.loader === 'vanilla' || !inst.loader) {
      return 'Bản sạch nguyên bản không nạp mod, độ ổn định tuyệt đối';
    }
    return `Minecraft ${inst.gameVersion} (${inst.loader?.toUpperCase() || 'Vanilla'}) tùy chỉnh độc lập`;
  };

  const iconOptions = [
    { id: 'cube', label: 'Khối Minecraft', icon: <Box className="w-4 h-4 text-emerald-400" /> },
    { id: 'flask', label: 'Bình thuốc', icon: <FlaskConical className="w-4 h-4 text-purple-400" /> },
    { id: 'swords', label: 'Gươm chiến', icon: <Swords className="w-4 h-4 text-rose-400" /> },
    { id: 'pickaxe', label: 'Cúp đào', icon: <Pickaxe className="w-4 h-4 text-amber-400" /> },
    { id: 'skull', label: 'Boss / Hardcore', icon: <Skull className="w-4 h-4 text-purple-300" /> },
    { id: 'zap', label: 'Tia sét FPS', icon: <Zap className="w-4 h-4 text-yellow-400" /> },
    { id: 'shield', label: 'Khiên bảo vệ', icon: <Shield className="w-4 h-4 text-blue-400" /> },
    { id: 'sparkles', label: 'Ma thuật', icon: <Sparkles className="w-4 h-4 text-cyan-400" /> },
  ];

  return (
    <main
      className="flex-1 overflow-y-auto flex flex-col select-none relative z-10"
      style={{ padding: '22px 24px 28px 24px', gap: '22px', boxSizing: 'border-box' }}
    >
      {/* Header Row */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-xl md:text-2xl font-black text-white flex items-center tracking-tight gap-2.5">
              <Layers className="w-6 h-6 text-emerald-400" />
              <span>Bản cài đặt cá nhân</span>
            </h1>
            <span className="status-pill font-mono font-extrabold text-[10px] px-2 py-0.5 rounded-md bg-amber-500/15 text-amber-300 border border-amber-500/30">
              BETA
            </span>
            <span className="status-pill font-mono font-bold text-xs px-2.5 py-0.5 rounded-full bg-emerald-500/15 text-emerald-300 border border-emerald-500/25">
              {instances.length} bản cài
            </span>
          </div>
          <p className="text-xs text-slate-400 mt-1">
            Instance cục bộ và modpack được quản lý độc lập trên ổ cứng của bạn
          </p>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2.5">
          <button
            onClick={handleManualRefresh}
            className="status-pill rounded-xl bg-white/[0.04] hover:bg-white/[0.08] border border-white/10 hover:border-white/20 text-slate-300 hover:text-white transition-all font-semibold cursor-pointer flex items-center gap-2"
            style={{ height: '38px', padding: '0 16px', fontSize: '12.5px' }}
            title="Quét lại thư mục instances để cập nhật"
          >
            <RotateCw className={`w-3.5 h-3.5 text-cyan-400 ${isRefreshing ? 'animate-spin' : ''}`} />
            <span>Quét lại máy</span>
          </button>

          <button
            onClick={() => {
              setShowCreateModal(true);
              setIsNameCustomized(false);
            }}
            className="status-pill rounded-xl bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 font-bold transition-all shadow-md shadow-emerald-500/20 cursor-pointer flex items-center gap-2"
            style={{ height: '38px', padding: '0 18px', fontSize: '13px' }}
          >
            <Plus className="w-4 h-4" />
            <span>Thêm bản cài mới</span>
          </button>
        </div>
      </div>

      {/* Grid of Instances */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
          gap: '20px',
        }}
      >
        {instances.map((inst) => {
          const isSelected = activeInstance?.id === inst.id;
          const badge = getInstanceBadge(inst);
          const desc = getInstanceDesc(inst);
          const loaderLabel = inst.loader ? inst.loader.toUpperCase() : 'VANILLA';
          const ramGb = Math.round(inst.maxMemoryMb / 1024);

          return (
            <div
              key={inst.id}
              style={{
                padding: '24px',
                borderRadius: '22px',
                gap: '16px',
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between',
              }}
              className={`glass-panel border transition-all ${
                isSelected
                  ? 'border-emerald-500/50 bg-emerald-500/[0.06] shadow-xl shadow-emerald-500/10 ring-1 ring-emerald-500/30'
                  : 'border-white/[0.08] hover:border-white/[0.18] bg-slate-900/60'
              }`}
            >
              {/* Card Top: Badges and Title */}
              <div className="flex flex-col gap-2.5">
                <div className="flex items-center justify-between">
                  <span
                    className={`status-pill font-mono font-bold text-[10px] tracking-wider uppercase px-2.5 py-0.5 rounded-md border ${badge.className}`}
                  >
                    {badge.label}
                  </span>
                  <span className="text-[11px] font-mono text-slate-400">
                    MC {inst.gameVersion}
                  </span>
                </div>

                <h3 className="font-extrabold text-lg text-white tracking-tight truncate">
                  {inst.name}
                </h3>

                <p className="text-xs text-slate-300 leading-relaxed line-clamp-2">
                  {desc}
                </p>
              </div>

              {/* Card Spec Details */}
              <div className="bg-black/35 rounded-xl border border-white/5 px-3.5 py-2.5 flex items-center justify-between text-xs">
                <div className="flex items-center gap-2 text-slate-400">
                  <HardDrive className="w-3.5 h-3.5 text-cyan-400" />
                  <span className="font-semibold text-slate-300 font-mono">
                    {loaderLabel} {inst.loaderVersion ? `(${inst.loaderVersion})` : ''}
                  </span>
                </div>
                <span className="font-mono text-emerald-400 font-bold">
                  {ramGb} GB RAM
                </span>
              </div>

              {/* Card Footer Actions */}
              <div className="flex items-center justify-between pt-1 border-t border-white/5 gap-2">
                <div className="flex items-center gap-1.5">
                  <button
                    onClick={() => OpenInstanceFolder(inst.id)}
                    className="p-2 rounded-xl bg-white/[0.04] hover:bg-white/[0.1] text-slate-400 hover:text-white transition cursor-pointer"
                    title="Mở thư mục instance"
                  >
                    <FolderOpen className="w-4 h-4 text-cyan-400" />
                  </button>

                  <button
                    onClick={() => onOpenSettings(inst)}
                    className="p-2 rounded-xl bg-white/[0.04] hover:bg-white/[0.1] text-slate-400 hover:text-white transition cursor-pointer"
                    title="Cấu hình instance"
                  >
                    <Settings className="w-4 h-4 text-slate-300" />
                  </button>

                  {instances.length > 1 && (
                    <button
                      onClick={() => handleDelete(inst.id, inst.name)}
                      className="p-2 rounded-xl bg-white/[0.04] hover:bg-rose-500/15 text-slate-400 hover:text-rose-400 transition cursor-pointer"
                      title="Xóa instance"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>

                <button
                  onClick={() => onSelectInstance(inst)}
                  className={`status-pill px-4 rounded-xl text-xs font-bold transition-all cursor-pointer flex items-center gap-1.5 ${
                    isSelected
                      ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/35 shadow-sm'
                      : 'bg-white/[0.06] hover:bg-emerald-500/15 text-slate-200 hover:text-emerald-300 border border-white/10 hover:border-emerald-500/30'
                  }`}
                  style={{ height: '36px' }}
                >
                  {isSelected ? (
                    <>
                      <Check className="w-3.5 h-3.5 text-emerald-400" />
                      <span>ĐANG CHỌN</span>
                    </>
                  ) : (
                    <span>Chọn</span>
                  )}
                </button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Bottom Info Banner */}
      <div className="glass-panel border border-white/10 rounded-2xl p-4 bg-slate-950/60 flex items-center justify-between flex-wrap gap-3 mt-2">
        <p className="text-xs text-slate-400">
          Các bản cài đặt cá nhân chạy độc lập và không ảnh hưởng đến cấu hình P2P server.
        </p>
        <button
          onClick={() => OpenFolder('')}
          className="text-xs font-bold text-emerald-400 hover:text-emerald-300 transition flex items-center gap-1 cursor-pointer"
        >
          <span>Mở thư mục gốc</span>
          <ExternalLink className="w-3.5 h-3.5" />
        </button>
      </div>

      {/* ======================================================== */}
      {/* TLAUNCHER / PRISM STYLE FULL CREATE INSTANCE MODAL */}
      {/* ======================================================== */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 bg-black/85 backdrop-blur-md flex items-center justify-center p-4 overflow-y-auto">
          <div
            className="glass-panel modal-enter w-full max-w-2xl border border-emerald-500/30 shadow-2xl relative rounded-3xl bg-slate-950/95 flex flex-col my-8"
            style={{ padding: '28px 32px', gap: '22px' }}
          >
            {/* Modal Header */}
            <div className="flex items-center justify-between border-b border-white/10 pb-4">
              <div className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-xl bg-emerald-500/15 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
                  <Plus className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-lg font-black text-white tracking-tight">
                    Tạo Bản Cài Đặt Mới (Instance)
                  </h2>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Hỗ trợ đầy đủ mọi phiên bản Minecraft, Fabric, Forge và NeoForge chuẩn TLauncher
                  </p>
                </div>
              </div>
              <button
                onClick={() => setShowCreateModal(false)}
                className="w-8 h-8 rounded-xl hover:bg-white/10 flex items-center justify-center text-slate-400 hover:text-white transition cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Modal Body */}
            <div className="flex flex-col gap-5 text-xs overflow-y-auto max-h-[70vh] pr-1">
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
                        ? 'border-emerald-500/50 bg-emerald-500/15 shadow-md shadow-emerald-500/10'
                        : 'border-white/10 bg-white/[0.02] hover:bg-white/[0.05] text-slate-400'
                    }`}
                  >
                    <div className="flex items-center justify-between w-full">
                      <Box className={`w-5 h-5 ${loader === 'vanilla' ? 'text-emerald-400' : 'text-slate-400'}`} />
                      {loader === 'vanilla' && <Check className="w-3.5 h-3.5 text-emerald-400" />}
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
                        ? 'border-cyan-500/50 bg-cyan-500/15 shadow-md shadow-cyan-500/10'
                        : 'border-white/10 bg-white/[0.02] hover:bg-white/[0.05] text-slate-400'
                    }`}
                  >
                    <div className="flex items-center justify-between w-full">
                      <Zap className={`w-5 h-5 ${loader === 'fabric' ? 'text-cyan-400' : 'text-slate-400'}`} />
                      {loader === 'fabric' && <Check className="w-3.5 h-3.5 text-cyan-400" />}
                    </div>
                    <div>
                      <div className="font-bold text-white text-xs">Fabric</div>
                      <div className="text-[10px] text-cyan-300/80 mt-0.5">Tối ưu FPS cao</div>
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
                          ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
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
                          ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
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
                          ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
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
                          ? 'bg-emerald-500 text-slate-950 font-extrabold shadow-sm'
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
                      {loadingLoaders && <Loader2 className="w-3 h-3 text-cyan-400 animate-spin" />}
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
                      <span className="status-pill px-2.5 py-1.5 rounded-xl font-mono text-[10.5px] font-bold bg-emerald-500/15 text-emerald-300 border border-emerald-500/30 shrink-0">
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
                      className="text-[10px] text-emerald-400 hover:text-emerald-300 flex items-center gap-1 cursor-pointer"
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
                    <Cpu className="w-3.5 h-3.5 text-cyan-400" />
                    <span>5. Phân Bổ Bộ Nhớ RAM</span>
                  </span>
                  <span className="font-mono text-emerald-400 font-extrabold text-sm">
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
                            ? 'bg-emerald-500 text-slate-950 font-bold'
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
                    <div className="w-7 h-7 rounded-lg bg-emerald-500/20 border border-emerald-500/30 flex items-center justify-center text-emerald-400 shrink-0">
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
            <div className="flex items-center justify-end pt-3 border-t border-white/10 gap-3">
              <button
                type="button"
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
        </div>
      )}
    </main>
  );
};
