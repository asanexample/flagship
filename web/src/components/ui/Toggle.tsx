import styles from "./Toggle.module.css";

interface ToggleProps {
  checked: boolean;
  onChange: (next: boolean) => void;
  /** Accessible name — required, since the visual switch has no text of its own. */
  label: string;
  disabled?: boolean;
  /** "danger" tints the ON state red — for high-stakes switches (e.g. a prod kill switch). */
  tone?: "accent" | "danger";
}

/** An accessible switch: a real button with role=switch + aria-checked, keyboard-operable, reduced-motion safe. */
export function Toggle({ checked, onChange, label, disabled = false, tone = "accent" }: ToggleProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      data-tone={tone}
      className={styles.track}
      onClick={() => onChange(!checked)}
    >
      <span className={styles.thumb} aria-hidden />
    </button>
  );
}
