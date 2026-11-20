import { useEffect, useRef, type ReactNode } from 'react';

interface DialogProps {
  titleId: string;
  onClose: () => void;
  busy?: boolean;
  children: ReactNode;
}

export function Dialog({ titleId, onClose, busy, children }: DialogProps) {
  const dialog = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const element = dialog.current!;
    element.showModal();
    return () => { element.close(); };
  }, []);

  return (
    <dialog ref={dialog} className="cam-dialog" aria-labelledby={titleId}
      onCancel={(event) => { event.preventDefault(); if (!busy) onClose(); }}>
      {children}
    </dialog>
  );
}
