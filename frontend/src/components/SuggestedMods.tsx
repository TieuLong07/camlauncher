import { useEffect, useState } from 'react';
import { Check, Download, Film, Loader2, RefreshCw, Trash2 } from 'lucide-react';
import { GetSuggestedMods } from '../../wailsjs/go/main/App';
import { models } from '../../wailsjs/go/models';

interface SuggestedModsProps {
  serverId: string;
  installed: boolean;
  busy: boolean;
  onChangeMod: (id: string, install: boolean) => Promise<void>;
  showToast: (message: string) => void;
}

export function SuggestedMods({ serverId, installed, busy, onChangeMod, showToast }: SuggestedModsProps) {
  const [mods, setMods] = useState<models.SuggestedMod[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [actionError, setActionError] = useState('');
  const [pending, setPending] = useState<{ id: string; install: boolean } | null>(null);
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    GetSuggestedMods(serverId).then((items) => {
      if (!cancelled) { setMods(items ?? []); setLoadError(''); }
    }).catch((reason) => {
      if (!cancelled) setLoadError(`Không đọc được mod tùy chỉnh. ${String(reason)}`);
    }).finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [serverId, installed, busy, retry]);

  const change = async (mod: models.SuggestedMod, install: boolean) => {
    setPending({ id: mod.id, install });
    setActionError('');
    try {
      await onChangeMod(mod.id, install);
      showToast(install ? `Đã ${mod.installed ? 'giữ' : 'cài'} ${mod.name} làm mod tùy chỉnh.` : `Đã gỡ ${mod.name}.`);
    } catch (reason) {
      setActionError(`Không thể ${install ? 'cài' : 'gỡ'} ${mod.name}. ${String(reason)}`);
    } finally {
      setPending(null);
      setRetry((value) => value + 1);
    }
  };

  return (
    <section className="suggested-mods" aria-labelledby="suggested-mods-title" aria-busy={Boolean(pending)}>
      <div className="suggested-mods-heading">
        <h3 id="suggested-mods-title">Mod tùy chỉnh gợi ý</h3>
        <span>Kamaron · Forge 1.20.1</span>
      </div>
      <p className="suggested-mods-intro">Chọn cài thêm theo cách bạn thích chơi. Các mod này chỉ dùng trên máy người chơi.</p>
      {!installed && <p className="suggested-mods-note">Cài modpack Kamaron trước để thêm mod tùy chỉnh.</p>}
      {loading && mods.length === 0 && <p className="suggested-mods-note" role="status">Đang kiểm tra mod trên máy…</p>}
      {(loadError || actionError) && <div className="suggested-mods-error" role="alert"><p>{loadError || actionError}</p>
        {loadError && <button className="cam-button" onClick={() => setRetry((value) => value + 1)} disabled={busy || Boolean(pending)}>
          <RefreshCw size={15} /> Kiểm tra lại
        </button>}
      </div>}
      <ul className="suggested-mods-list">
        {mods.map((mod) => {
          const changing = pending?.id === mod.id;
          const disabled = !installed || busy || loading || Boolean(pending) || mod.modified || Boolean(loadError);
          return <li key={mod.id} className="suggested-mod-row">
            <Film className="suggested-mod-icon" size={26} aria-hidden="true" />
            <div className="suggested-mod-copy">
              <div className="suggested-mod-title"><h4>{mod.name}</h4>
                {mod.installed && <span className={mod.modified ? '' : 'is-ready'}><Check size={13} /> {mod.modified ? 'Tệp đã thay đổi' : 'Đã cài'}</span>}
              </div>
              <p>{mod.description}</p>
              <small>v{mod.version} · {(mod.size / (1024 * 1024)).toFixed(0)} MB · Chỉ phía client</small>
              {mod.modified && <p className="suggested-mods-note">Tệp trên máy khác bản gợi ý. Kiểm tra trong thư mục mods trước khi thay đổi.</p>}
              {mod.installed && mod.managed && <p className="suggested-mods-note">Bản cũ cần cập nhật để kết nối server không cài Flashback.</p>}
            </div>
            <div className="suggested-mod-actions">
              {changing ? <span className="suggested-mods-note" role="status"><Loader2 size={16} className="animate-spin" /> {pending.install ? 'Đang cài…' : 'Đang gỡ…'}</span>
                : <>
                  {(!mod.installed || mod.managed) && <button className="cam-button" disabled={disabled} onClick={() => void change(mod, true)} aria-label={`${mod.installed ? 'Giữ' : 'Cài thêm'} ${mod.name}`}>
                    <Download size={16} /> {mod.installed ? 'Cập nhật mod' : 'Cài thêm'}
                  </button>}
                  {mod.installed && <button className="cam-button" disabled={disabled} onClick={() => void change(mod, false)} aria-label={`Gỡ ${mod.name}`}><Trash2 size={16} /> Gỡ</button>}
                </>}
            </div>
          </li>;
        })}
      </ul>
      {!loading && !loadError && mods.length === 0 && <p className="suggested-mods-note">Chưa có mod gợi ý cho máy chủ này.</p>}
      {installed && busy && !pending && <p className="suggested-mods-note">Chờ tác vụ hiện tại hoàn tất và đóng Minecraft trước khi thay đổi mod.</p>}
    </section>
  );
}
