import { useEffect, useRef, type ReactNode } from "react";
import styles from "./Dialog.module.css";

/** A modal built on the native <dialog> — free focus-trap, Escape-to-close, and backdrop. */
export function Dialog({
  open,
  onClose,
  title,
  children,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
}) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    else if (!open && el.open) el.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      className={styles.dialog}
      onClose={onClose}
      onClick={(e) => {
        // Clicking the backdrop (the dialog element itself, outside the panel) closes.
        if (e.target === ref.current) onClose();
      }}
    >
      <div className={styles.panel}>
        <header className={styles.head}>
          <h2 className={styles.title}>{title}</h2>
          <button type="button" className={styles.close} onClick={onClose} aria-label="Close dialog">
            ✕
          </button>
        </header>
        <div className={styles.body}>{children}</div>
      </div>
    </dialog>
  );
}
