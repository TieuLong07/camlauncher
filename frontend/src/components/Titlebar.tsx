import { Download, Minus, Square, X } from 'lucide-react';
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime';
import { models } from '../../wailsjs/go/models';
import orangeLogo from '../assets/images/cam-orange.png';

interface TitlebarProps {
  appVersion: string;
  launcherUpdate: models.LauncherUpdateInfo | null;
  onOpenLauncherUpdate: () => void;
}

export function Titlebar({ appVersion, launcherUpdate, onOpenLauncherUpdate }: TitlebarProps) {
  return (
    <header className="launcher-titlebar wails-drag">
      <div className="launcher-brand">
        <img src={orangeLogo} width="32" height="32" alt="" />
        <strong>CamLauncher</strong>
        {appVersion && <span>{appVersion}</span>}
      </div>
      <div className="launcher-window-actions wails-no-drag">
        {launcherUpdate?.hasUpdate && (
          <button className="cam-button titlebar-update" onClick={onOpenLauncherUpdate}>
            <Download size={15} /> Có bản cập nhật
          </button>
        )}
        <button onClick={WindowMinimise} title="Thu nhỏ" aria-label="Thu nhỏ"><Minus size={16} /></button>
        <button onClick={WindowToggleMaximise} title="Đổi kích thước cửa sổ" aria-label="Đổi kích thước cửa sổ"><Square size={13} /></button>
        <button onClick={Quit} title="Đóng launcher" aria-label="Đóng launcher"><X size={17} /></button>
      </div>
    </header>
  );
}
