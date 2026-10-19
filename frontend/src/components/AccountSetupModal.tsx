import React, { useState, useEffect } from 'react';
import { User, Sparkles, AlertCircle, Check, ArrowRight } from 'lucide-react';
import { models } from '../../wailsjs/go/models';
import { AddOfflineAccount, RemoveAccount, SetActiveAccount } from '../../wailsjs/go/main/App';

interface AccountSetupModalProps {
  isOpen: boolean;
  existingAccounts: models.Account[];
  onAccountCreated: (account: models.Account) => void;
  onOpenMicrosoftLogin: () => void;
  showToast: (msg: string) => void;
}

export const AccountSetupModal: React.FC<AccountSetupModalProps> = ({
  isOpen,
  existingAccounts,
  onAccountCreated,
  onOpenMicrosoftLogin,
  showToast,
}) => {
  const [username, setUsername] = useState('');
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');
  const [avatarName, setAvatarName] = useState('Steve');

  useEffect(() => {
    const timer = setTimeout(() => {
      const trimmed = username.trim();
      setAvatarName(trimmed && isValidName(trimmed) ? trimmed : 'Steve');
    }, 400);
    return () => clearTimeout(timer);
  }, [username]);

  if (!isOpen) return null;

  const isValidName = (name: string): boolean => {
    return /^[a-zA-Z0-9_]{3,16}$/.test(name);
  };

  const isCamPlayer = (name: string): boolean => {
    return name.trim().toLowerCase() === 'camplayer';
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const cleanName = username.trim();

    if (!cleanName) {
      setErrorMsg('Vui lòng nhập tên người chơi');
      return;
    }

    if (isCamPlayer(cleanName)) {
      setErrorMsg('Không được sử dụng tên mặc định CamPlayer. Vui lòng chọn tên riêng của bạn.');
      return;
    }

    if (!isValidName(cleanName)) {
      setErrorMsg('Tên từ 3-16 ký tự, chỉ gồm chữ, số và dấu gạch dưới (_)');
      return;
    }

    setLoading(true);
    setErrorMsg('');

    try {
      const newAcc = await AddOfflineAccount(cleanName);
      if (newAcc && newAcc.id) {
        await SetActiveAccount(newAcc.id);

        // Clean up any old default "CamPlayer" accounts to keep account list tidy
        const camPlayerAccounts = existingAccounts.filter(
          (a) => a.username.toLowerCase() === 'camplayer'
        );
        for (const cp of camPlayerAccounts) {
          try {
            await RemoveAccount(cp.id);
          } catch (_) {}
        }

        showToast(`Chào mừng ${cleanName} tham gia CamLauncher!`);
        onAccountCreated(newAcc);
      }
    } catch (err: any) {
      setErrorMsg(err.toString());
    } finally {
      setLoading(false);
    }
  };

  const nameValid = isValidName(username.trim()) && !isCamPlayer(username.trim());

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-xl select-none">
      <div className="modal-enter glass-panel border border-emerald-500/30 rounded-3xl max-w-md w-full p-8 shadow-2xl relative overflow-hidden flex flex-col items-center text-center">
        {/* Glow ambient circle */}
        <div className="absolute -top-24 left-1/2 -translate-x-1/2 w-48 h-48 bg-emerald-500/20 rounded-full blur-3xl pointer-events-none" />

        {/* Live Minotar Avatar Preview */}
        <div className="relative mb-5">
          <div className="w-20 h-20 rounded-2xl bg-slate-900/90 border-2 border-emerald-500/40 shadow-lg shadow-emerald-500/20 p-1 flex items-center justify-center overflow-hidden">
            <img
              src={`https://minotar.net/helm/${avatarName}/72.png`}
              alt="Skin Preview"
              className="w-full h-full object-cover pixelated rounded-xl transition-all duration-300"
              onError={(e) => {
                (e.target as HTMLImageElement).src = 'https://minotar.net/helm/Steve/72.png';
              }}
            />
          </div>
          <span className="absolute -bottom-1 -right-1 w-4 h-4 rounded-full bg-emerald-400 border-2 border-[#0a0d14]" />
        </div>

        {/* Heading */}
        <div className="flex items-center gap-2 mb-1.5">
          <Sparkles className="w-5 h-5 text-emerald-400" />
          <h2 className="text-2xl font-black text-white tracking-tight">
            Thiết Lập Tên Nhân Vật
          </h2>
        </div>

        <p className="text-xs text-slate-300 leading-relaxed mb-6 max-w-sm">
          Chào mừng bạn đến với <strong>CamLauncher</strong>! Vui lòng đặt tên nhân vật (IGN) của bạn để vào máy chủ Chill Town và lưu giữ hành trang cá nhân.
        </p>

        {/* Form */}
        <form onSubmit={handleSubmit} className="w-full flex flex-col gap-4 text-left">
          <div>
            <label className="block text-xs font-bold text-slate-200 mb-2 flex items-center justify-between">
              <span>Tên nhân vật (IGN)</span>
              <span className="text-[10px] text-slate-400 font-normal">3 - 16 ký tự</span>
            </label>

            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                <User className="w-4 h-4" />
              </div>

              <input
                type="text"
                value={username}
                onChange={(e) => {
                  setUsername(e.target.value);
                  setErrorMsg('');
                }}
                autoFocus
                placeholder="Ví dụ: TieuLong, Steve, BanhMi..."
                maxLength={16}
                className="w-full bg-white/[0.04] border border-white/10 focus:border-emerald-500/60 focus:bg-white/[0.07] rounded-xl pl-10 pr-10 py-3 text-sm font-semibold text-white placeholder-slate-500 focus:outline-none transition-all"
              />

              {nameValid && (
                <div className="absolute inset-y-0 right-0 pr-3.5 flex items-center pointer-events-none text-emerald-400">
                  <Check className="w-4 h-4" />
                </div>
              )}
            </div>

            {errorMsg && (
              <div className="mt-2 text-xs text-rose-400 flex items-center gap-1.5 bg-rose-500/10 border border-rose-500/20 px-3 py-1.5 rounded-lg">
                <AlertCircle className="w-3.5 h-3.5 shrink-0" />
                <span>{errorMsg}</span>
              </div>
            )}
          </div>

          {/* Primary CTA */}
          <button
            type="submit"
            disabled={loading || !username.trim()}
            className={`w-full py-3.5 rounded-xl font-black text-sm tracking-wide flex items-center justify-center gap-2 cursor-pointer btn-tactile ${
              nameValid && !loading
                ? 'bg-gradient-to-r from-emerald-500 to-teal-400 hover:from-emerald-400 hover:to-teal-300 text-slate-950 glow-primary shadow-lg shadow-emerald-500/25'
                : 'bg-white/[0.08] text-slate-500 cursor-not-allowed border border-white/5'
            }`}
          >
            <span>{loading ? 'Đang Thiết Lập...' : 'Bắt Đầu Chơi Ngay'}</span>
            <ArrowRight className="w-4 h-4" />
          </button>
        </form>

        {/* Divider */}
        <div className="relative w-full my-5 flex items-center justify-center">
          <div className="w-full border-t border-white/10" />
          <span className="absolute bg-[#0f141e] px-3 text-[10px] font-mono text-slate-400 uppercase tracking-widest">
            HOẶC
          </span>
        </div>

        {/* Microsoft Option */}
        <button
          type="button"
          onClick={onOpenMicrosoftLogin}
          className="w-full py-2.5 px-4 rounded-xl bg-white/[0.03] hover:bg-blue-600/15 border border-white/10 hover:border-blue-500/40 text-slate-300 hover:text-blue-300 text-xs font-bold flex items-center justify-center gap-2.5 transition-all btn-tactile cursor-pointer"
        >
          <svg className="w-4 h-4 shrink-0" viewBox="0 0 21 21">
            <rect x="1" y="1" width="9" height="9" fill="#f25022" />
            <rect x="11" y="1" width="9" height="9" fill="#7fba00" />
            <rect x="1" y="11" width="9" height="9" fill="#00a4ef" />
            <rect x="11" y="11" width="9" height="9" fill="#ffb900" />
          </svg>
          <span>Đăng nhập tài khoản Microsoft (Bản quyền)</span>
        </button>
      </div>
    </div>
  );
};
