import React, { useState, useEffect, useRef } from 'react';
import {
  Package,
  Search,
  Download,
  CheckCircle2,
  Sparkles,
} from 'lucide-react';
import { models } from '../../wailsjs/go/models';
import { SearchMods, InstallModToInstance, InstallPackToInstance } from '../../wailsjs/go/main/App';

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
  const [loading, setLoading] = useState(true);
  const [searchError, setSearchError] = useState(false);
  const [installError, setInstallError] = useState('');
  const searchRequest = useRef(0);
  const [installingKey, setInstallingKey] = useState<string | null>(null);
  const [installedKeys, setInstalledKeys] = useState<Record<string, boolean>>({});

  const doSearch = async (searchTerm: string, type: string) => {
    const request = ++searchRequest.current;
    setLoading(true);
    setSearchError(false);
    try {
      const version = activeInstance?.gameVersion || '1.21.1';
      const loader = type === 'mod' ? (activeInstance ? activeInstance.loader || 'vanilla' : 'fabric') : '';
      const result = await SearchMods(searchTerm, version, loader, type);
      if (request === searchRequest.current) setResults(result?.hits || []);
    } catch {
      if (request === searchRequest.current) setSearchError(true);
    } finally {
      if (request === searchRequest.current) setLoading(false);
    }
  };

  useEffect(() => {
    // Initial search for popular mods whenever projectType or activeInstance changes
    doSearch(query, projectType);
    return () => { searchRequest.current += 1; };
  }, [projectType, activeInstance?.id, activeInstance?.gameVersion, activeInstance?.loader]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    doSearch(query, projectType);
  };

  const handleInstall = async (project: models.ModrinthProject) => {
    if (!activeInstance || installingKey) return;
    const key = `${activeInstance.id}:${projectType}:${project.slug}`;
    setInstallError('');
    setInstallingKey(key);
    try {
      if (projectType === 'mod') {
        await InstallModToInstance(activeInstance.id, project.slug);
      } else {
        await InstallPackToInstance(activeInstance.id, project.slug, projectType);
      }
      setInstalledKeys((previous) => ({ ...previous, [key]: true }));
    } catch (error) {
      setInstallError(`Không thể cài ${project.title} vào ${activeInstance.name}. ${error}`);
    } finally {
      setInstallingKey(null);
    }
  };

  return (
    <main
      className="flex-1 overflow-y-auto flex flex-col select-none relative z-10"
      style={{ padding: '20px 18px 24px 18px', gap: '22px', boxSizing: 'border-box' }}
    >
      <div className="instance-header">
        <div><h1>Kho Mods</h1><p>Mod, shader và gói tài nguyên từ Modrinth</p></div>
        <label className="mod-target-label">
          Cài vào
          <select value={activeInstance?.id ?? ''} disabled={instances.length === 0 || Boolean(installingKey)}
            onChange={(event) => {
              const chosen = instances.find((instance) => instance.id === event.target.value);
              if (chosen) onSelectInstance?.(chosen);
            }}>
            {instances.length === 0 && <option value="">Chưa có bản cài</option>}
            {instances.map((instance) => (
              <option key={instance.id} value={instance.id}>
                {instance.name} · {instance.gameVersion} · {instance.loader || 'Vanilla'}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="glass-panel rounded-xl border border-white/10 p-4 grid gap-3.5">
        <form aria-label="Tìm trong Kho Mods" onSubmit={handleSearchSubmit} className="flex items-center" style={{ gap: '12px' }}>
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
            <input
              type="text"
              aria-label="Tên mod, shader hoặc gói tài nguyên" placeholder="Tìm kiếm mod, shader (Sodium, Iris, JEI, JourneyMap, Physics, Complementary...)..."
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
                      ? 'bg-orange-500/20 text-orange-300 border border-orange-500/40'
                      : 'bg-white/[0.04] text-slate-300 hover:text-white hover:bg-white/[0.08] border border-white/10'
                  }`}
                  style={{ height: '34px', padding: '0 14px', gap: '8px', fontSize: '12px' }}
                >
                  <span>{cat.label}</span>
                  <span
                    className={`text-[10px] font-mono rounded-md px-1.5 py-0.5 ${
                      isCurrent ? 'bg-orange-500/30 text-orange-200' : 'bg-black/30 text-slate-400'
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
            <span>Đang tìm cho MC {activeInstance?.gameVersion || '1.21.1'} ({activeInstance ? activeInstance.loader || 'Vanilla' : 'Fabric'})</span>
          </div>
        </div>

      </div>
      {projectType === 'shader' && (
        <p className="text-sm text-amber-200">Shader cần Iris hoặc OptiFine trong bản cài để sử dụng.</p>
      )}
      {installError && <p className="text-sm text-rose-300" role="alert">{installError}</p>}

      {/* Results Grid */}
      {loading ? (
        <div
          className="flex flex-col items-center justify-center py-24"
          style={{ gap: '14px' }}
        >
          <div className="w-9 h-9 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin" />
          <p className="text-sm text-slate-300" role="status">Đang tìm trên Modrinth…</p>
        </div>
      ) : searchError ? (
        <div className="flex flex-col items-center gap-3 py-12">
          <p role="alert" className="text-sm text-rose-300">Không thể tải kết quả từ Modrinth.</p>
          <button className="cam-button" onClick={() => doSearch(query, projectType)}>Thử lại</button>
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
            const key = `${activeInstance?.id}:${projectType}:${item.slug}`;
            const isInstalled = installedKeys[key];
            const isBusy = installingKey === key;

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
                    disabled={!activeInstance || Boolean(installingKey) || isInstalled}
                    className={`status-pill w-full rounded-xl text-xs font-bold transition-all cursor-pointer ${
                      isInstalled
                        ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 shadow-sm shadow-emerald-500/10'
                        : isBusy
                        ? 'bg-emerald-500/30 text-emerald-200 border border-emerald-500/30 cursor-wait'
                        : 'bg-white/[0.06] hover:bg-orange-400 text-slate-100 hover:text-slate-950 border border-white/10 hover:border-orange-400 hover:shadow-lg hover:shadow-emerald-500/20'
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
