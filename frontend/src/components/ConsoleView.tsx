import React, { useState, useEffect, useRef } from 'react';
import {
  Terminal,
  Trash2,
  Copy,
  Check,
  AlertTriangle,
  Flame,
  ArrowDown,
  Sparkles,
} from 'lucide-react';
import { LogEvent } from '../types';

interface ConsoleViewProps {
  logs: LogEvent[];
  onClearLogs: () => void;
  isRunning: boolean;
}

export const ConsoleView: React.FC<ConsoleViewProps> = ({ logs, onClearLogs, isRunning }) => {
  const [autoScroll, setAutoScroll] = useState(true);
  const [filterLevel, setFilterLevel] = useState<string>('ALL');
  const [copied, setCopied] = useState(false);
  const logEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (autoScroll) {
      logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  const handleCopy = () => {
    const text = logs.map((l) => `[${l.timestamp}] [${l.level}] ${l.line}`).join('\n');
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Smart Crash Log Analyzer
  const analyzeCrash = () => {
    const text = logs.map((l) => l.line).join('\n');
    if (text.includes('OutOfMemoryError') || text.includes('Java heap space')) {
      return {
        title: 'Tràn bộ nhớ RAM (OutOfMemoryError)',
        solution: 'Game bị thiếu RAM! Hãy vào mục "Phiên Bản", chỉnh tăng lượng RAM phân bổ lên ít nhất 4096MB (4GB) hoặc 6144MB (6GB).',
        severity: 'danger',
      };
    }
    if (text.includes('UnsupportedClassVersionError') || text.includes('has been compiled by a more recent version of the Java Runtime')) {
      return {
        title: 'Sai phiên bản Java (Java Version Mismatch)',
        solution: 'Minecraft 1.20.5+ yêu cầu Java 21, 1.17-1.20.4 yêu cầu Java 17, còn <=1.16.5 cần Java 8. Hãy cài đặt bản Java phù hợp trong mục Cài Đặt.',
        severity: 'danger',
      };
    }
    if (text.includes('net.fabricmc.loader.impl.FormattedException: Mod resolution encountered an incompatible mod set')) {
      return {
        title: 'Xung đột phiên bản Mod (Fabric Mod Incompatibility)',
        solution: 'Có mod yêu cầu phiên bản Fabric hoặc Minecraft khác, hoặc thiếu "Fabric API". Hãy vào Kho Mod tải Fabric API hoặc gỡ mod bị xung đột.',
        severity: 'warning',
      };
    }
    if (text.includes('Mixin apply failed')) {
      return {
        title: 'Lỗi nạp mã nguồn Mod (Mixin Apply Failed)',
        solution: 'Hai mod đang cố gắng thay đổi cùng một hàm đồ họa của Minecraft (thường xảy ra khi dùng OptiFine chung với Sodium). Hãy chỉ dùng Sodium + Iris.',
        severity: 'warning',
      };
    }
    return null;
  };

  const crashDiagnosis = analyzeCrash();

  const filteredLogs = logs.filter((l) => {
    if (filterLevel === 'ALL') return true;
    return l.level === filterLevel;
  });

  return (
    <div
      className="flex-1 flex flex-col overflow-hidden"
      style={{ padding: '20px 18px 24px 18px', gap: '18px', boxSizing: 'border-box' }}
    >
      {/* Top Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0">
        <div>
          <h1 className="text-2xl font-extrabold text-white flex items-center gap-2.5">
            <Terminal className="w-6 h-6 text-emerald-400" />
            Nhật Ký & Console Log
          </h1>
          <p className="text-xs text-slate-400 mt-1">
            Theo dõi luồng stdout/stderr trực tiếp từ Java và chẩn đoán lỗi crash thông minh.
          </p>
        </div>

        <div className="flex items-center gap-2">
          {/* Level Filter */}
          <div className="flex items-center bg-black/40 rounded-xl p-1 border border-white/10 text-xs">
            {['ALL', 'INFO', 'WARN', 'ERROR'].map((lvl) => (
              <button
                key={lvl}
                onClick={() => setFilterLevel(lvl)}
                className={`px-2.5 py-1 rounded-lg font-semibold transition cursor-pointer ${
                  filterLevel === lvl
                    ? 'bg-emerald-500/20 text-emerald-300 font-bold'
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                {lvl}
              </button>
            ))}
          </div>

          <button
            onClick={() => setAutoScroll(!autoScroll)}
            className={`p-2 rounded-xl border text-xs font-semibold flex items-center gap-1 cursor-pointer transition ${
              autoScroll
                ? 'bg-emerald-500/20 border-emerald-500/30 text-emerald-300'
                : 'bg-white/5 border-white/10 text-slate-400'
            }`}
            title="Tự động cuộn theo log mới"
          >
            <ArrowDown className="w-3.5 h-3.5" />
            <span>Cuộn</span>
          </button>

          <button
            onClick={handleCopy}
            className="p-2 rounded-xl bg-white/5 hover:bg-white/10 border border-white/10 text-slate-300 text-xs font-semibold flex items-center gap-1 cursor-pointer transition"
            title="Sao chép toàn bộ log"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
            <span>{copied ? 'Đã Chép' : 'Copy'}</span>
          </button>

          <button
            onClick={onClearLogs}
            className="p-2 rounded-xl bg-white/5 hover:bg-rose-500/20 border border-white/10 text-slate-300 hover:text-rose-300 text-xs font-semibold flex items-center gap-1 cursor-pointer transition"
            title="Xóa nhật ký"
          >
            <Trash2 className="w-3.5 h-3.5" />
            <span>Xóa</span>
          </button>
        </div>
      </div>

      {/* Crash Diagnosis Banner if found */}
      {crashDiagnosis && (
        <div className="p-4 rounded-xl bg-rose-950/40 border border-rose-500/40 backdrop-blur-md flex items-start gap-3 shrink-0">
          <AlertTriangle className="w-5 h-5 text-rose-400 shrink-0 mt-0.5" />
          <div className="space-y-1 text-xs">
            <div className="font-extrabold text-sm text-rose-200">
              Phát hiện nguyên nhân lỗi: {crashDiagnosis.title}
            </div>
            <p className="text-slate-300 leading-relaxed">{crashDiagnosis.solution}</p>
          </div>
        </div>
      )}

      {/* Log Terminal Window */}
      <div className="flex-1 bg-[#050811] rounded-2xl border border-white/[0.08] p-4 overflow-y-auto font-mono text-xs space-y-1 shadow-inner select-text">
        {filteredLogs.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-slate-600 space-y-2">
            <Terminal className="w-8 h-8 stroke-1" />
            <p className="font-sans">Chưa có nhật ký nào. Hãy bấm "Vào Game" để theo dõi luồng máy ảo Java.</p>
          </div>
        ) : (
          filteredLogs.map((item, idx) => {
            let colorClass = 'text-slate-300';
            if (item.level === 'WARN') colorClass = 'text-amber-300';
            if (item.level === 'ERROR') colorClass = 'text-rose-400 font-semibold';
            if (item.level === 'CHAT') colorClass = 'text-purple-300';

            return (
              <div key={idx} className="leading-relaxed hover:bg-white/[0.02] px-1 rounded flex gap-2">
                <span className="text-slate-600 select-none shrink-0">[{item.timestamp}]</span>
                <span
                  className={`select-none font-bold shrink-0 ${
                    item.level === 'ERROR'
                      ? 'text-rose-400'
                      : item.level === 'WARN'
                      ? 'text-amber-400'
                      : 'text-emerald-500'
                  }`}
                >
                  [{item.level}]
                </span>
                <span className={`${colorClass} break-all`}>{item.line}</span>
              </div>
            );
          })
        )}
        <div ref={logEndRef} />
      </div>
    </div>
  );
};
