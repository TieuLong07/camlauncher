import React, { useState, useEffect, useRef } from 'react';
import {
  Package,
  Search,
  Download,
  CheckCircle2,
  Sparkles,
  ChevronDown,
  Check,
  Layers,
} from 'lucide-react';
import { models } from '../../wailsjs/go/models';
import { SearchMods, InstallModToInstance } from '../../wailsjs/go/main/App';

interface ModStoreViewProps {
  activeInstance: models.Instance | null;
  instances?: models.Instance[];
  onSelectInstance?: (instance: models.Instance) => void;
}

export const ModStoreView: React.FC<ModStoreViewProps> = ({
  activeInstance,
  instances = [],
  onSelectInstance,
}) => {
  const [query, setQuery] = useState('');
  const [projectType, setProjectType] = useState('mod'); // 'mod' | 'shader' | 'resourcepack'
  const [results, setResults] = useState<models.ModrinthProject[]>([]);
  const [loading, setLoading] = useState(false);
  const [installingSlug, setInstallingSlug] = useState<string | null>(null);
  const [installedSlugs, setInstalledSlugs] = useState<Record<string, boolean>>({});
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setDropdownOpen(false);
      }
    };
    if (dropdownOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [dropdownOpen]);

  const doSearch = async (searchTerm: string, type: string) => {
    setLoading(true);
    try {
      const version = activeInstance?.gameVersion || '1.21.1';
      const loader = activeInstance?.loader || 'fabric';
      const res = await SearchMods(searchTerm, version, loader, type);
      setResults(res?.hits || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    // Initial search for popular mods whenever projectType or activeInstance changes
    doSearch(query, projectType);
  }, [projectType, activeInstance]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    doSearch(query, projectType);
  };

  const handleInstall = async (project: models.ModrinthProject) => {
    if (!activeInstance) {
      alert('Vui lòng chọn một phiên bản (instance) trước khi cài mod!');
      return;
    }
    setInstallingSlug(project.slug);
    try {
      await InstallModToInstance(activeInstance.id, project.slug);
      setInstalledSlugs((prev) => ({ ...prev, [project.slug]: true }));
    } catch (err: any) {
      alert(`Lỗi cài đặt mod ${project.title}: ${err}`);
    } finally {
      setInstallingSlug(null);
    }
  };

  return (
    <main
      className="flex-1 overflow-y-auto flex flex-col select-none relative z-10"
      style={{ padding: '20px 18px 24px 18px', gap: '22px', boxSizing: 'border-box' }}
    >
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between" style={{ gap: '14px' }}>
        <div>
          <div className="flex items-center" style={{ gap: '10px' }}>
            <h1 className="text-2xl font-black text-white flex items-center tracking-tight" style={{ gap: '10px' }}>
              <Package className="w-6 h-6 text-emerald-400" />
              Kho Mod & Shaders Modrinth
            </h1>
            <span className="status-pill font-mono font-extrabold text-[10px] px-2 py-0.5 rounded-md bg-amber-500/15 text-amber-300 border border-amber-500/30">
              BETA
            </span>
            <span
              className="status-pill rounded-full font-mono font-semibold bg-emerald-500/15 text-emerald-300 border border-emerald-500/25"
              style={{ height: '24px', padding: '0 10px', fontSize: '10.5px' }}
            >
              MODRINTH API
            </span>
          </div>
          <p className="text-xs text-slate-400 mt-1">
            Khám phá hàng chục ngàn Mod, Shader và Resource Pack tối ưu hóa. Cài đặt tự động chỉ với 1 click.
          </p>
        </div>

        {/* Instance Selector Dropdown */}
        <div className="relative self-start md:self-auto" ref={dropdownRef}>
          <button
            type="button"
            onClick={() => setDropdownOpen(!dropdownOpen)}
            className={`status-pill rounded-xl transition-all cursor-pointer border ${
              dropdownOpen
                ? 'bg-emerald-500/15 border-emerald-500/40 text-white shadow-lg shadow-emerald-500/10'
                : 'bg-white/[0.04] hover:bg-white/[0.08] border-white/[0.08] hover:border-white/20 text-slate-200'
            }`}
            style={{ height: '40px', padding: '0 16px', gap: '10px' }}
            title="Bấm để chọn phiên bản (instance) cài đặt mod"
          >
            <div className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            <span className="text-slate-400 text-xs font-normal">Đích cài đặt:</span>
            {activeInstance ? (
              <>
                <span className="font-bold text-emerald-300 text-xs">{activeInstance.name}</span>
                <span
                  className="status-pill rounded-md bg-emerald-500/15 text-emerald-300 border border-emerald-500/25 font-mono"
                  style={{ height: '22px', padding: '0 8px', fontSize: '10.5px' }}
                >
                  MC {activeInstance.gameVersion} • {activeInstance.loader?.toUpperCase() || 'VANILLA'}
                </span>
              </>
            ) : (
              <span className="text-amber-300 font-semibold text-xs">Chưa chọn instance</span>
            )}
            <ChevronDown
              className={`w-4 h-4 text-slate-400 transition-transform duration-200 ${
                dropdownOpen ? 'rotate-180 text-emerald-300' : ''
              }`}
            />
          </button>

          {/* Dropdown Menu */}
          {dropdownOpen && (
            <div
              className="absolute right-0 top-full mt-2 z-50 glass-panel border border-white/15 rounded-2xl shadow-2xl p-2 modal-enter"
              style={{
                width: '340px',
                backgroundColor: 'rgba(12, 16, 26, 0.95)',
                backdropFilter: 'blur(20px)',
              }}
            >
              <div className="px-3 py-2 border-b border-white/10 flex items-center justify-between">
                <span className="text-[11px] font-bold tracking-wider text-slate-400 uppercase">
                  Chọn phiên bản (Instance)
                </span>
                <span className="text-[10px] text-emerald-400 font-mono font-semibold">
                  {instances.length} phiên bản
                </span>
              </div>

              <div className="max-h-64 overflow-y-auto py-1 flex flex-col" style={{ gap: '4px' }}>
                {instances.length === 0 ? (
                  <div className="py-6 text-center text-xs text-slate-500">
                    Chưa có instance nào. Hãy tạo ở trang Chơi Game!
                  </div>
                ) : (
                  instances.map((inst) => {
                    const isSelected = activeInstance?.id === inst.id;
                    return (
                      <button
                        key={inst.id}
                        type="button"
                        onClick={() => {
                          if (onSelectInstance) {
                            onSelectInstance(inst);
                          }
                          setDropdownOpen(false);
                        }}
                        className={`w-full flex items-center justify-between p-2.5 rounded-xl transition-all cursor-pointer text-left ${
                          isSelected
                            ? 'bg-emerald-500/15 border border-emerald-500/35 text-white shadow-sm'
                            : 'hover:bg-white/[0.06] text-slate-300 hover:text-white border border-transparent'
                        }`}
                      >
                        <div className="flex items-center min-w-0" style={{ gap: '10px' }}>
                          <div className="w-8 h-8 rounded-lg bg-white/5 border border-white/10 flex items-center justify-center shrink-0">
                            <Layers className="w-4 h-4 text-emerald-400" />
                          </div>
                          <div className="min-w-0">
                            <div className="font-bold text-xs truncate flex items-center" style={{ gap: '6px' }}>
                              <span className="truncate">{inst.name}</span>
                              {inst.serverSyncUrl && (
                                <span className="status-pill text-[9px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-mono">
                                  SYNC
                                </span>
                              )}
                            </div>
                            <div className="text-[10px] text-slate-400 mt-0.5 flex items-center" style={{ gap: '6px' }}>
                              <span>MC {inst.gameVersion}</span>
                              <span>•</span>
                              <span className="uppercase text-emerald-400 font-mono font-semibold">
                                {inst.loader || 'Vanilla'}
                              </span>
                            </div>
                          </div>
                        </div>

                        {isSelected && (
                          <Check className="w-4 h-4 text-emerald-400 shrink-0 ml-2" />
                        )}
                      </button>
                    );
                  })
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Search & Filter Bar */}
      <div
        className="glass-panel"
        style={{
          padding: '16px 20px',
          borderRadius: '18px',
          display: 'flex',
          flexDirection: 'column',
          gap: '14px',
          border: '1px solid rgba(255, 255, 255, 0.08)',
        }}
      >
        <form onSubmit={handleSearchSubmit} className="flex items-center" style={{ gap: '12px' }}>
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
            <input
              type="text"
              placeholder="Tìm kiếm mod, shader (Sodium, Iris, JEI, JourneyMap, Physics, Complementary...)..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="w-full pl-10 pr-4 rounded-xl bg-black/40 border border-white/10 text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500/50 text-xs font-medium transition-colors"
              style={{ height: '42px' }}
            />
          </div>
          <button
            type="submit"
            className="status-pill rounded-xl bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 font-bold transition-all shadow-md shadow-emerald-500/20 cursor-pointer"
            style={{ height: '42px', padding: '0 22px', gap: '8px', fontSize: '13px' }}
          >
            <Search className="w-4 h-4" />
            <span>Tìm Kiếm</span>
          </button>
        </form>

        {/* Categories & API info */}
        <div className="flex flex-wrap items-center justify-between" style={{ gap: '10px', paddingTop: '2px' }}>
          <div className="flex items-center flex-wrap" style={{ gap: '8px' }}>
            <span className="text-slate-400 text-xs font-semibold mr-1">Danh mục:</span>
            {[
              { id: 'mod', label: 'Mods Game', badge: 'Gameplay & FPS' },
              { id: 'shader', label: 'Shaders Pack', badge: 'Iris & Opti' },
              { id: 'resourcepack', label: 'Resource Packs', badge: 'Textures' },
            ].map((cat) => {
              const isCurrent = projectType === cat.id;
              return (
                <button
                  key={cat.id}
                  type="button"
                  onClick={() => setProjectType(cat.id)}
                  className={`status-pill rounded-xl transition-all cursor-pointer font-semibold ${
                    isCurrent
                      ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 shadow-sm shadow-emerald-500/15'
                      : 'bg-white/[0.04] text-slate-300 hover:text-white hover:bg-white/[0.08] border border-white/10'
                  }`}
                  style={{ height: '34px', padding: '0 14px', gap: '8px', fontSize: '12px' }}
                >
                  <span>{cat.label}</span>
                  <span
                    className={`text-[10px] font-mono rounded-md px-1.5 py-0.5 ${
                      isCurrent ? 'bg-emerald-500/30 text-emerald-200' : 'bg-black/30 text-slate-400'
                    }`}
                  >
                    {cat.badge}
                  </span>
                </button>
              );
            })}
          </div>

          <div className="text-[11px] text-slate-400 font-mono hidden md:flex items-center" style={{ gap: '6px' }}>
            <Sparkles className="w-3.5 h-3.5 text-emerald-400" />
            <span>Đang tìm cho MC {activeInstance?.gameVersion || '1.21.1'} ({activeInstance?.loader?.toUpperCase() || 'FABRIC'})</span>
          </div>
        </div>

        {/* Quick Switch Instance Pill Row */}
        {instances.length > 1 && (
          <div
            className="flex items-center flex-wrap"
            style={{ gap: '8px', paddingTop: '10px', borderTop: '1px solid rgba(255, 255, 255, 0.05)' }}
          >
            <span className="text-slate-400 text-xs font-semibold mr-1 flex items-center" style={{ gap: '6px' }}>
              <Layers className="w-3.5 h-3.5 text-emerald-400" />
              <span>Chuyển nhanh đích:</span>
            </span>
            {instances.map((inst) => {
              const isSelected = activeInstance?.id === inst.id;
              return (
                <button
                  key={inst.id}
                  type="button"
                  onClick={() => onSelectInstance && onSelectInstance(inst)}
                  className={`status-pill rounded-lg transition-all cursor-pointer font-medium ${
                    isSelected
                      ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 shadow-sm shadow-emerald-500/10'
                      : 'bg-white/[0.03] text-slate-400 hover:text-white hover:bg-white/[0.06] border border-white/5'
                  }`}
                  style={{ height: '28px', padding: '0 10px', gap: '6px', fontSize: '11px' }}
                >
                  <span className="font-semibold">{inst.name}</span>
                  <span className="text-[10px] text-slate-400 font-mono">
                    ({inst.loader || 'Vanilla'} {inst.gameVersion})
                  </span>
                  {isSelected && <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />}
                </button>
              );
            })}
          </div>
        )}
      </div>

      {/* Results Grid */}
      {loading ? (
        <div
          className="flex flex-col items-center justify-center py-24"
          style={{ gap: '14px' }}
        >
          <div className="w-9 h-9 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin" />
          <p className="text-xs text-slate-400 font-medium">Đang tìm kiếm kho lưu trữ Modrinth...</p>
        </div>
      ) : results.length === 0 ? (
        <div
          className="flex flex-col items-center justify-center py-24 text-slate-500"
          style={{ gap: '10px' }}
        >
          <Package className="w-12 h-12 stroke-1 text-slate-600" />
          <p className="text-sm font-bold text-slate-300">Không tìm thấy mod nào phù hợp</p>
          <p className="text-xs text-slate-500">Hãy thử từ khóa khác hoặc chuyển danh mục Shaders / Resource Packs</p>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(310px, 1fr))', gap: '18px' }}>
          {results.map((item) => {
            const isInstalled = installedSlugs[item.slug];
            const isBusy = installingSlug === item.slug;

            return (
              <div
                key={item.project_id}
                style={{
                  padding: '20px 22px',
                  borderRadius: '18px',
                  display: 'flex',
                  flexDirection: 'column',
                  justifyContent: 'space-between',
                  gap: '16px',
                  boxSizing: 'border-box',
                  background: 'rgba(15, 23, 42, 0.65)',
                }}
                className="glass-panel border border-white/[0.08] hover:border-emerald-500/40 hover:bg-slate-900/80 transition-all group"
              >
                <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  {/* Icon + Title + Author Header */}
                  <div className="flex items-start" style={{ gap: '14px' }}>
                    {item.icon_url ? (
                      <img
                        src={item.icon_url}
                        alt={item.title}
                        className="w-12 h-12 rounded-xl object-cover bg-black/40 border border-white/10 shrink-0 shadow-md group-hover:scale-105 transition-transform"
                      />
                    ) : (
                      <div className="w-12 h-12 rounded-xl bg-white/5 border border-white/10 flex items-center justify-center shrink-0">
                        <Package className="w-6 h-6 text-emerald-400/80" />
                      </div>
                    )}
                    <div className="min-w-0 flex-1">
                      <h3 className="font-extrabold text-sm text-white group-hover:text-emerald-300 transition-colors truncate">
                        {item.title}
                      </h3>
                      <p className="text-[11px] text-slate-400 mt-0.5 truncate">
                        Tác giả: <span className="text-slate-300 font-medium">{item.author}</span>
                      </p>
                    </div>
                  </div>

                  {/* Description */}
                  <p className="text-xs text-slate-300/90 line-clamp-2 leading-relaxed" style={{ minHeight: '36px' }}>
                    {item.description || 'Không có mô tả chi tiết từ tác giả.'}
                  </p>

                  {/* Badges / Tags */}
                  <div className="flex flex-wrap items-center justify-between" style={{ gap: '8px', paddingTop: '4px' }}>
                    <div className="flex flex-wrap items-center" style={{ gap: '6px' }}>
                      {item.categories?.slice(0, 2).map((cat) => (
                        <span
                          key={cat}
                          className="status-pill rounded-md bg-white/[0.05] text-slate-300 uppercase font-mono font-medium border border-white/5"
                          style={{ height: '22px', padding: '0 8px', fontSize: '10px' }}
                        >
                          {cat}
                        </span>
                      ))}
                    </div>
                    <span
                      className="status-pill rounded-md bg-emerald-500/10 text-emerald-400 font-mono font-semibold border border-emerald-500/20"
                      style={{ height: '22px', padding: '0 8px', fontSize: '11px', gap: '4px' }}
                    >
                      <Download className="w-3 h-3" />
                      {item.downloads.toLocaleString()}
                    </span>
                  </div>
                </div>

                {/* Install Button Footer */}
                <div style={{ paddingTop: '12px', borderTop: '1px solid rgba(255, 255, 255, 0.08)' }}>
                  <button
                    onClick={() => handleInstall(item)}
                    disabled={isBusy}
                    className={`status-pill w-full rounded-xl text-xs font-bold transition-all cursor-pointer ${
                      isInstalled
                        ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 shadow-sm shadow-emerald-500/10'
                        : isBusy
                        ? 'bg-emerald-500/30 text-emerald-200 border border-emerald-500/30 cursor-wait'
                        : 'bg-white/[0.06] hover:bg-emerald-500 text-slate-100 hover:text-slate-950 border border-white/10 hover:border-emerald-400 hover:shadow-lg hover:shadow-emerald-500/20'
                    }`}
                    style={{ height: '40px', gap: '8px', fontSize: '12.5px' }}
                  >
                    {isBusy ? (
                      <>
                        <div className="w-4 h-4 border-2 border-emerald-400 border-t-transparent rounded-full animate-spin" />
                        <span>Đang Tải & Cài Đặt...</span>
                      </>
                    ) : isInstalled ? (
                      <>
                        <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                        <span>Đã Cài Vào {activeInstance ? activeInstance.name : 'Instance'}</span>
                      </>
                    ) : (
                      <>
                        <Download className="w-4 h-4" />
                        <span>Cài Vào {activeInstance ? activeInstance.name : 'Instance'}</span>
                      </>
                    )}
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </main>
  );
};
