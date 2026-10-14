import React, { useState } from 'react';
import {
  Users,
  UserPlus,
  Check,
  Trash2,
  ShieldCheck,
  X,
  User,
  Copy,
  ExternalLink,
  Loader2,
  AlertCircle,
} from 'lucide-react';
import { models, auth } from '../../wailsjs/go/models';
import {
  AddOfflineAccount,
  SetActiveAccount,
  RemoveAccount,
  StartMicrosoftAuth,
  PollMicrosoftAuth,
  CancelMicrosoftAuth,
} from '../../wailsjs/go/main/App';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

interface AccountsViewProps {
  accounts: models.Account[];
  activeAccount: models.Account | null;
  onRefresh: () => void;
}

export const AccountsView: React.FC<AccountsViewProps> = ({
  accounts,
  activeAccount,
  onRefresh,
}) => {
  const [showAddOffline, setShowAddOffline] = useState(false);
  const [username, setUsername] = useState('');
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  // Microsoft Device Code Flow States
  const [showMsModal, setShowMsModal] = useState(false);
  const [msLoading, setMsLoading] = useState(false);
  const [msDeviceCode, setMsDeviceCode] = useState<auth.DeviceCodeResult | null>(null);
  const [msPolling, setMsPolling] = useState(false);
  const [msError, setMsError] = useState('');
  const [msSuccess, setMsSuccess] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleAddOffline = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg('');
    if (!username.trim()) {
      setErrorMsg('Vui lòng nhập tên người chơi');
      return;
    }

    setLoading(true);
    try {
      await AddOfflineAccount(username.trim());
      setUsername('');
      setShowAddOffline(false);
      onRefresh();
    } catch (err: any) {
      setErrorMsg(err.toString());
    } finally {
      setLoading(false);
    }
  };

  const handleStartMicrosoftLogin = async () => {
    setMsError('');
    setMsSuccess(false);
    setMsLoading(true);
    setMsDeviceCode(null);
    setShowMsModal(true);

    try {
      const codeRes = await StartMicrosoftAuth();
      setMsDeviceCode(codeRes);
      setMsLoading(false);
      setMsPolling(true);

      // Begin background polling
      try {
        await PollMicrosoftAuth(codeRes.deviceCode, codeRes.interval, codeRes.expiresIn);
        setMsSuccess(true);
        setMsPolling(false);
        onRefresh();
        setTimeout(() => {
          setShowMsModal(false);
          setMsSuccess(false);
        }, 1800);
      } catch (pollErr: any) {
        setMsPolling(false);
        setMsError(pollErr.toString());
      }
    } catch (err: any) {
      setMsLoading(false);
      setMsError(err.toString());
    }
  };

  const handleCancelMicrosoft = async () => {
    try {
      await CancelMicrosoftAuth();
    } catch (_) {}
    setShowMsModal(false);
    setMsPolling(false);
    setMsLoading(false);
    setMsDeviceCode(null);
    setMsError('');
    setMsSuccess(false);
  };

  const handleCopyCode = () => {
    if (msDeviceCode?.userCode) {
      navigator.clipboard.writeText(msDeviceCode.userCode);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleOpenBrowser = () => {
    const url = msDeviceCode?.verificationUri || 'https://www.microsoft.com/link';
    BrowserOpenURL(url);
  };

  const handleSelect = async (id: string) => {
    await SetActiveAccount(id);
    onRefresh();
  };

  const handleDelete = async (id: string, name: string) => {
    if (confirm(`Bạn có chắc muốn xóa tài khoản "${name}"?`)) {
      await RemoveAccount(id);
      onRefresh();
    }
  };

  return (
    <main
      className="flex-1 overflow-y-auto flex flex-col select-none relative z-10"
      style={{ padding: '20px 18px 24px 18px', gap: '22px', boxSizing: 'border-box' }}
    >
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <h1 className="text-2xl font-black text-white flex items-center tracking-tight" style={{ gap: '10px' }}>
            <Users className="w-6 h-6 text-emerald-400" />
            Quản Lý Tài Khoản
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Chuyển đổi 1-click giữa tài khoản Offline (chơi miễn phí) và tài khoản Microsoft bản quyền.
          </p>
        </div>

        <div className="flex items-center gap-3">
          {/* Microsoft Login Button */}
          <button
            onClick={handleStartMicrosoftLogin}
            className="status-pill rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white font-bold transition-all shadow-md shadow-blue-500/20 cursor-pointer flex items-center"
            style={{ height: '40px', padding: '0 18px', gap: '8px', fontSize: '13px' }}
          >
            <svg className="w-4 h-4 shrink-0" viewBox="0 0 21 21">
              <rect x="1" y="1" width="9" height="9" fill="#f25022" />
              <rect x="11" y="1" width="9" height="9" fill="#7fba00" />
              <rect x="1" y="11" width="9" height="9" fill="#00a4ef" />
              <rect x="11" y="11" width="9" height="9" fill="#ffb900" />
            </svg>
            <span>Đăng Nhập Microsoft Auth</span>
          </button>

          {/* Add Offline Button */}
          <button
            onClick={() => setShowAddOffline(true)}
            className="status-pill rounded-xl bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 font-bold transition-all shadow-md shadow-emerald-500/20 cursor-pointer flex items-center"
            style={{ height: '40px', padding: '0 18px', gap: '8px', fontSize: '13px' }}
          >
            <UserPlus className="w-4 h-4" />
            <span>Thêm Nick Offline (Crack)</span>
          </button>
        </div>
      </div>

      {/* Grid of Accounts */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(290px, 1fr))', gap: '18px' }}>
        {accounts.map((acc) => {
          const isActive = activeAccount?.id === acc.id;
          const isMicrosoft = acc.type === 'microsoft';

          return (
            <div
              key={acc.id}
              style={{
                padding: '20px 22px',
                borderRadius: '18px',
                gap: '16px',
                display: 'flex',
                flexDirection: 'column',
              }}
              className={`glass-panel border transition-all ${
                isActive
                  ? isMicrosoft
                    ? 'border-blue-500/50 bg-blue-500/[0.06] shadow-xl shadow-blue-500/10'
                    : 'border-emerald-500/50 bg-emerald-500/[0.06] shadow-xl shadow-emerald-500/10'
                  : 'border-white/[0.08] hover:border-white/[0.18] bg-slate-900/60'
              }`}
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center" style={{ gap: '14px' }}>
                  <div className="relative">
                    <img
                      src={`https://minotar.net/helm/${acc.username}/48.png`}
                      alt={acc.username}
                      className="w-12 h-12 rounded-xl pixelated bg-black/40 border border-white/15 shadow-md"
                      onError={(e) => {
                        (e.target as HTMLElement).style.display = 'none';
                      }}
                    />
                    <div className="w-12 h-12 rounded-xl bg-white/5 border border-white/10 hidden items-center justify-center">
                      <User className="w-6 h-6 text-emerald-400" />
                    </div>
                  </div>

                  <div>
                    <h3 className="font-extrabold text-base text-white">{acc.username}</h3>
                    <div className="flex items-center mt-1">
                      {isMicrosoft ? (
                        <span
                          className="status-pill font-mono font-bold uppercase tracking-wider rounded-md bg-blue-500/15 text-blue-300 border border-blue-500/25 flex items-center gap-1"
                          style={{ height: '22px', padding: '0 8px', fontSize: '10px' }}
                        >
                          <ShieldCheck className="w-3 h-3 text-blue-400" />
                          <span>Microsoft Auth</span>
                        </span>
                      ) : (
                        <span
                          className="status-pill font-mono font-bold uppercase tracking-wider rounded-md bg-emerald-500/15 text-emerald-300 border border-emerald-500/25"
                          style={{ height: '22px', padding: '0 8px', fontSize: '10px' }}
                        >
                          Offline / Crack
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {accounts.length > 1 && (
                  <button
                    onClick={() => handleDelete(acc.id, acc.username)}
                    className="p-2 rounded-lg text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 transition cursor-pointer"
                    title="Xóa tài khoản"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                )}
              </div>

              {/* UUID */}
              <div
                className="bg-black/30 rounded-xl border border-white/5 text-[11px] flex flex-col"
                style={{ padding: '12px 14px', gap: '4px' }}
              >
                <div className="text-slate-400 font-medium">
                  {isMicrosoft ? 'UUID (Mojang Online):' : 'UUID (Mojang v3 Offline):'}
                </div>
                <div className="font-mono text-slate-300 truncate">{acc.uuid}</div>
              </div>

              {/* Action */}
              <button
                onClick={() => handleSelect(acc.id)}
                className={`status-pill w-full rounded-xl text-xs font-bold transition cursor-pointer ${
                  isActive
                    ? isMicrosoft
                      ? 'bg-blue-500/20 text-blue-300 border border-blue-500/35'
                      : 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/35'
                    : 'bg-white/[0.05] hover:bg-emerald-500/15 text-slate-200 hover:text-emerald-300 border border-white/10 hover:border-emerald-500/30'
                }`}
                style={{ height: '42px', gap: '8px', fontSize: '12.5px' }}
              >
                {isActive ? (
                  <>
                    <Check className={`w-4 h-4 ${isMicrosoft ? 'text-blue-400' : 'text-emerald-400'}`} />
                    <span>Đang Sử Dụng</span>
                  </>
                ) : (
                  <span>Chọn Tài Khoản Này</span>
                )}
              </button>
            </div>
          );
        })}
      </div>

      {/* Microsoft Device Code Flow Modal */}
      {showMsModal && (
        <div className="fixed inset-0 z-50 bg-black/85 backdrop-blur-md flex items-center justify-center p-4">
          <div
            className="glass-panel modal-enter w-full max-w-lg border border-blue-500/30 shadow-2xl relative bg-slate-950/90"
            style={{ padding: '30px', borderRadius: '24px', gap: '22px', display: 'flex', flexDirection: 'column' }}
          >
            {/* Modal Header */}
            <div className="flex items-center justify-between border-b border-white/10 pb-3">
              <div className="flex items-center gap-2.5">
                <svg className="w-5 h-5 shrink-0" viewBox="0 0 21 21">
                  <rect x="1" y="1" width="9" height="9" fill="#f25022" />
                  <rect x="11" y="1" width="9" height="9" fill="#7fba00" />
                  <rect x="1" y="11" width="9" height="9" fill="#00a4ef" />
                  <rect x="11" y="11" width="9" height="9" fill="#ffb900" />
                </svg>
                <h2 className="text-lg font-bold text-white tracking-tight">Đăng Nhập Microsoft Auth</h2>
              </div>
              <button
                onClick={handleCancelMicrosoft}
                className="w-7 h-7 rounded-lg hover:bg-white/10 flex items-center justify-center text-slate-400 hover:text-white transition-colors cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Modal Content */}
            {msLoading && (
              <div className="flex flex-col items-center justify-center py-8 gap-3">
                <Loader2 className="w-8 h-8 text-blue-400 animate-spin" />
                <p className="text-sm font-semibold text-slate-300">Đang khởi tạo mã xác nhận từ Microsoft...</p>
              </div>
            )}

            {msSuccess && (
              <div className="flex flex-col items-center justify-center py-8 gap-3">
                <div className="w-12 h-12 rounded-full bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center">
                  <Check className="w-6 h-6 text-emerald-400" />
                </div>
                <h3 className="text-base font-bold text-white">Đăng Nhập Thành Công!</h3>
                <p className="text-xs text-slate-400">Đã đồng bộ hồ sơ Minecraft bản quyền vào launcher.</p>
              </div>
            )}

            {!msLoading && !msSuccess && msDeviceCode && (
              <div className="flex flex-col gap-5">
                {/* Step 1 */}
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-bold text-slate-300 uppercase tracking-wider">
                    Bước 1: Mở liên kết xác thực trên trình duyệt
                  </span>
                  <div className="flex items-center gap-2 bg-black/40 border border-white/10 rounded-xl px-3.5 py-2.5">
                    <span className="text-xs font-mono text-blue-300 truncate flex-1">
                      {msDeviceCode.verificationUri}
                    </span>
                    <button
                      onClick={handleOpenBrowser}
                      className="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold transition flex items-center gap-1.5 cursor-pointer shrink-0 shadow-sm shadow-blue-500/30"
                    >
                      <ExternalLink className="w-3.5 h-3.5" />
                      <span>Mở Trình Duyệt</span>
                    </button>
                  </div>
                </div>

                {/* Step 2 */}
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-bold text-slate-300 uppercase tracking-wider">
                    Bước 2: Nhập mã thiết bị bên dưới
                  </span>
                  <div className="flex items-center justify-between bg-blue-950/40 border border-blue-500/30 rounded-2xl p-4">
                    <div className="flex flex-col">
                      <span className="text-[10px] text-blue-300/70 font-semibold tracking-wider uppercase">
                        MÃ XÁC THỰC (CODE)
                      </span>
                      <span className="text-2xl font-black font-mono tracking-widest text-white mt-0.5">
                        {msDeviceCode.userCode}
                      </span>
                    </div>
                    <button
                      onClick={handleCopyCode}
                      className={`px-4 py-2.5 rounded-xl font-bold text-xs transition flex items-center gap-1.5 cursor-pointer ${
                        copied
                          ? 'bg-emerald-500 text-slate-950 shadow-md shadow-emerald-500/20'
                          : 'bg-white/10 hover:bg-white/20 text-white border border-white/15'
                      }`}
                    >
                      {copied ? (
                        <>
                          <Check className="w-4 h-4" />
                          <span>Đã Chép</span>
                        </>
                      ) : (
                        <>
                          <Copy className="w-4 h-4" />
                          <span>Sao Chép</span>
                        </>
                      )}
                    </button>
                  </div>
                </div>

                {/* Polling Indicator */}
                {msPolling && (
                  <div className="flex items-center gap-3 p-3.5 rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-300 text-xs">
                    <Loader2 className="w-4 h-4 animate-spin text-blue-400 shrink-0" />
                    <span>Đang chờ bạn xác nhận và cấp quyền trên trình duyệt...</span>
                  </div>
                )}
              </div>
            )}

            {/* Error Message */}
            {msError && (
              <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs flex items-start gap-2.5">
                <AlertCircle className="w-4 h-4 text-rose-400 shrink-0 mt-0.5" />
                <div className="flex-1">{msError}</div>
              </div>
            )}

            {/* Modal Actions */}
            <div className="flex items-center justify-end pt-2 border-t border-white/5 gap-2.5">
              {msError ? (
                <>
                  <button
                    type="button"
                    onClick={handleCancelMicrosoft}
                    className="status-pill px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white transition cursor-pointer"
                    style={{ height: '36px', padding: '0 16px' }}
                  >
                    Đóng
                  </button>
                  <button
                    type="button"
                    onClick={handleStartMicrosoftLogin}
                    className="status-pill px-5 py-2 text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white rounded-xl transition cursor-pointer"
                    style={{ height: '36px', padding: '0 18px' }}
                  >
                    Thử Lại
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  onClick={handleCancelMicrosoft}
                  className="status-pill px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white transition cursor-pointer"
                  style={{ height: '36px', padding: '0 16px' }}
                >
                  Hủy Thao Tác
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Add Offline Modal */}
      {showAddOffline && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div
            className="glass-panel modal-enter w-full max-w-md border border-white/15 shadow-2xl relative"
            style={{ padding: '28px', borderRadius: '24px', gap: '20px', display: 'flex', flexDirection: 'column' }}
          >
            <div className="flex items-center justify-between border-b border-white/10 pb-3">
              <h2 className="text-lg font-bold text-white flex items-center" style={{ gap: '8px' }}>
                <UserPlus className="w-5 h-5 text-emerald-400" />
                Thêm Nick Offline (Crack)
              </h2>
              <button
                onClick={() => setShowAddOffline(false)}
                className="w-7 h-7 rounded-lg hover:bg-white/10 flex items-center justify-center text-slate-400 hover:text-white transition-colors cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <form onSubmit={handleAddOffline} className="flex flex-col" style={{ gap: '16px' }}>
              <div className="flex flex-col" style={{ gap: '6px' }}>
                <label className="text-slate-300 text-xs font-semibold">Tên nhân vật (Nickname):</label>
                <input
                  type="text"
                  placeholder="Ví dụ: Steve, Alex, TiCaGamer..."
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  maxLength={16}
                  autoFocus
                  className="w-full px-3.5 py-2.5 rounded-xl bg-black/40 border border-white/10 text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500/50 text-sm font-semibold"
                />
                <p className="text-[11px] text-slate-400 mt-1">
                  Từ 3 đến 16 ký tự. Tự động sinh UUID chuẩn Mojang tương thích 100% với các máy chủ Offline / Crack.
                </p>
              </div>

              {errorMsg && (
                <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs">
                  {errorMsg}
                </div>
              )}

              <div className="flex items-center justify-end pt-2" style={{ gap: '10px' }}>
                <button
                  type="button"
                  onClick={() => setShowAddOffline(false)}
                  className="status-pill px-4 py-2 text-xs font-semibold text-slate-400 hover:text-white transition-colors cursor-pointer"
                  style={{ height: '36px', padding: '0 16px' }}
                >
                  Hủy
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="status-pill px-5 py-2 text-xs font-bold bg-emerald-500 hover:bg-emerald-400 text-slate-950 rounded-xl transition-colors cursor-pointer disabled:opacity-50"
                  style={{ height: '36px', padding: '0 20px' }}
                >
                  {loading ? 'Đang Tạo...' : 'Xác Nhận'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </main>
  );
};
