import { useEffect, useRef, useState } from 'react';
import { Box, ChevronDown, Copy, Download, Radio, RefreshCw, Sprout, Truck, Zap } from 'lucide-react';
import { GetKamaronLiveStatus, GetServerLiveStatus } from '../../wailsjs/go/main/App';
import { models, tunnel } from '../../wailsjs/go/models';
import { RECOMMENDED_SERVERS, RecommendedServerItem } from '../data/servers';
import chillTownBanner from '../assets/images/chill-town-banner.webp';
import kamaronBanner from '../assets/images/kamaron-banner.webp';

interface PlayViewProps {
  selectedServer: RecommendedServerItem;
  onSelectServer: (server: RecommendedServerItem) => void;
  tunnelStatus: tunnel.TunnelStatus | null;
  tunnelBusy: boolean;
  onToggleTunnel: () => void;
  packStatus: string;
  checkingPack: boolean;
  onCheckPack: () => void;
  installed: boolean;
  busy: boolean;
  needsJava: boolean;
  updateResult?: models.UpdateCheckResult;
  showToast: (message: string) => void;
}

export function PlayView({
  selectedServer, onSelectServer, tunnelStatus, tunnelBusy, onToggleTunnel,
  packStatus, checkingPack, onCheckPack, installed, busy, needsJava, updateResult, showToast,
}: PlayViewProps) {
  const [liveStatus, setLiveStatus] = useState<models.ServerLiveStatus | null>(null);
  const [checkingPing, setCheckingPing] = useState(false);
  const [pingError, setPingError] = useState(false);
  const refreshPing = useRef<() => void>(() => {});
  const isKamaron = Boolean(selectedServer.isKamaron);
  const isChillTown = selectedServer.id === 'chilltown';
  const hasStatus = isChillTown || isKamaron;
  const tunnelInactive = !tunnelStatus?.running || !tunnelStatus.serverTarget?.endsWith(`:${selectedServer.serverPort}`);
  const address = !tunnelInactive ? `127.0.0.1:${tunnelStatus?.localPort}` : 'Bật Cam Tunnel để lấy địa chỉ';

  useEffect(() => {
    let cancelled = false;
    let pending = false;
    setLiveStatus(null);
    setPingError(false);
    setCheckingPing(false);
    const poll = async (manual = false) => {
      if (cancelled || !hasStatus) return;
      if (manual) setCheckingPing(true);
      if (pending) return;
      pending = true;
      try {
        const status = await (isKamaron ? GetKamaronLiveStatus() : GetServerLiveStatus());
        if (!cancelled) {
          setLiveStatus(status);
          setPingError(false);
        }
      } catch {
        if (!cancelled) {
          setLiveStatus(null);
          setPingError(true);
        }
      } finally {
        pending = false;
        if (!cancelled) setCheckingPing(false);
      }
    };
    refreshPing.current = () => { void poll(true); };
    void poll();
    const timer = setInterval(poll, 5000);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [isKamaron, hasStatus, tunnelInactive]);

  const connectionTitle = !hasStatus ? 'Chưa có thông tin trạng thái' : checkingPing ? 'Đang kiểm tra…'
    : liveStatus?.online ? 'Đang hoạt động'
      : tunnelInactive ? 'Cam Tunnel chưa bật'
        : pingError || liveStatus?.online === false ? 'Không liên lạc được' : 'Đang kiểm tra…';
  const connectionDetail = liveStatus?.online ? `${liveStatus.players}/${liveStatus.maxPlayers} người chơi · ${liveStatus.pingMs} ms`
    : tunnelInactive ? 'CamLauncher sẽ kết nối khi chơi.'
      : 'Kết nối qua Cam Tunnel.';

  const copyAddress = async () => {
    try {
      await navigator.clipboard.writeText(address);
      showToast('Đã sao chép địa chỉ máy chủ.');
    } catch {
      showToast('Không thể sao chép địa chỉ máy chủ.');
    }
  };

  return (
    <main className="server-home">
      <div className="server-home-inner">
        <h1>Máy chủ</h1>
        <div className="server-picker" role="group" aria-label="Chọn máy chủ">
          {RECOMMENDED_SERVERS.map((server) => {
            const SelectedIcon = server.isKamaron ? Truck : Sprout;
            const selected = server.id === selectedServer.id;
            return (
              <button key={server.id} className="server-choice" aria-pressed={selected} onClick={() => onSelectServer(server)}>
                <span className="server-choice-icon"><SelectedIcon size={20} /></span>
                <span><strong>{server.title}</strong>
                  <small>{server.id === 'chilltown' ? 'Ẩm thực & cơ giới hóa' : server.isKamaron ? 'Xe cộ & nông trại' : server.subtitle}</small></span>
                {selected && <span className="choice-indicator" aria-hidden="true" />}
              </button>
            );
          })}
        </div>

        <section className="server-hero" aria-labelledby="server-title"
          style={{ backgroundImage: `linear-gradient(90deg, #15100ff5 0%, #15100fb0 36%, #15100f12 76%), url(${isKamaron ? kamaronBanner : isChillTown ? chillTownBanner : selectedServer.bannerUrl})` }}>
          <div className="server-hero-copy">
            <div className="server-hero-version"><Box size={15} /> {updateResult?.manifest ? `${updateResult.manifest.loader} ${updateResult.manifest.gameVersion} · v${updateResult.latestVersion}` : selectedServer.version}</div>
            <h2 id="server-title">{selectedServer.title}</h2>
            <p>{isKamaron
              ? 'Mở rộng nông trại, lái xe và giao thương trong thế giới sinh tồn Lẩu Cam.'
              : isChillTown ? 'Xây nông trại, nấu ăn và tự động hóa cùng bạn bè trong thế giới sinh tồn Chill Town.' : selectedServer.description}</p>
          </div>
        </section>

        <div className="server-readiness" aria-label="Tình trạng máy chủ và bản cài">
          <section className="readiness-card">
            <div className="readiness-icon"><Radio size={20} /></div>
            <div className="readiness-copy">
              <h3>Kết nối máy chủ</h3>
              <strong className={liveStatus?.online ? 'is-ready' : ''} role="status">{connectionTitle}</strong>
              <p>{connectionDetail}</p>
            </div>
            {hasStatus && <button className="cam-button cam-icon" onClick={() => refreshPing.current()}
              title="Kiểm tra kết nối" aria-label="Kiểm tra kết nối" disabled={checkingPing}>
              <RefreshCw size={17} className={checkingPing ? 'animate-spin' : ''} />
            </button>}
          </section>
          <section className="readiness-card">
            <div className="readiness-icon"><Download size={20} /></div>
            <div className="readiness-copy">
              <h3>Bản cài trên máy</h3>
              <strong className={packStatus === 'Modpack đã sẵn sàng' ? 'is-ready' : ''} role="status">{packStatus}</strong>
              <p>{needsJava ? 'Cần cài Java 17.' : updateResult?.manifest?.expectedMods?.length ? `${updateResult.manifest.expectedMods.length} mod chính thức` : 'Kiểm tra phiên bản và tệp modpack.'}</p>
            </div>
            {installed && <button className="cam-button cam-icon" onClick={onCheckPack}
              title="Kiểm tra cập nhật modpack" aria-label="Kiểm tra cập nhật modpack" disabled={checkingPack || busy}>
              <RefreshCw size={17} className={checkingPack ? 'animate-spin' : ''} />
            </button>}
          </section>
        </div>

        {((updateResult?.missingMods?.length ?? 0) > 0 || (updateResult?.modifiedMods?.length ?? 0) > 0 || (updateResult?.extraMods?.length ?? 0) > 0 || (updateResult?.filesToDelete?.length ?? 0) > 0) && (
          <details className="server-details">
            <summary><ChevronDown size={16} /> Chi tiết mod
              {' · '}{updateResult?.missingMods?.length ?? 0} thiếu · {updateResult?.modifiedMods?.length ?? 0} thay đổi · {updateResult?.extraMods?.length ?? 0} mod cá nhân
            </summary>
            <div className="server-details-content mod-drift">
              {(updateResult?.missingMods?.length ?? 0) > 0 && <div><strong>Mod cần khôi phục</strong>
                <ul>{updateResult?.missingMods?.map((name) => <li key={name}>{name}</li>)}</ul></div>}
              {(updateResult?.modifiedMods?.length ?? 0) > 0 && <div><strong>Mod đã thay đổi nội dung</strong>
                <ul>{updateResult?.modifiedMods?.map((name) => <li key={name}>{name}</li>)}</ul></div>}
              {(updateResult?.extraMods?.length ?? 0) > 0 && <div><strong>Mod cá nhân được giữ lại</strong>
                <ul>{updateResult?.extraMods?.map((name) => <li key={name}>{name}</li>)}</ul></div>}
              {(updateResult?.filesToDelete?.length ?? 0) > 0 && <div><strong>Tệp chính thức của phiên bản cũ sẽ được xóa</strong>
                <ul>{updateResult?.filesToDelete?.map((name) => <li key={name}>{name}</li>)}</ul></div>}
            </div>
          </details>
        )}

        <details className="server-details">
          <summary><ChevronDown size={16} /> Chi tiết kết nối</summary>
          <div className="server-details-content">
            <button className="cam-button" onClick={copyAddress}><Copy size={16} /> {address}</button>
            {hasStatus && <button className="cam-button" onClick={onToggleTunnel} disabled={tunnelBusy}
              aria-pressed={!tunnelInactive}>
              <Zap size={16} /> {tunnelBusy ? 'Đang đổi kết nối…' : !tunnelInactive ? 'Tắt Cam Tunnel' : 'Bật Cam Tunnel'}
            </button>}
            <span>{'Địa chỉ cục bộ dùng khi Cam Tunnel đang bật.'}</span>
          </div>
        </details>
      </div>
    </main>
  );
}
