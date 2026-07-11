import type { ReactNode } from "react";
import styles from "./Badge.module.css";

type Tone = "on" | "off" | "partial" | "neutral" | "accent";

export function Badge({ tone = "neutral", children }: { tone?: Tone; children: ReactNode }) {
  return <span className={`${styles.badge} ${styles[tone]}`}>{children}</span>;
}
