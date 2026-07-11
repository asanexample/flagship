import type { EnvConfig } from "./types";

export function relativeTime(iso?: string): string {
  if (!iso) return "";
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const s = Math.round((Date.now() - t) / 1000);
  if (s < 60) return "just now";
  const m = Math.round(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.round(m / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.round(h / 24)}d ago`;
}

export type StateKind = "none" | "off" | "on" | "partial" | "targeted";

export interface StateSummary {
  kind: StateKind;
  label: string;
}

/** Reduce a per-Environment config to a single scannable state — the heart of the env strip. */
export function summarize(cfg: EnvConfig | null | undefined): StateSummary {
  if (!cfg) return { kind: "none", label: "—" };
  if (!cfg.enabled) return { kind: "off", label: "Off" };

  const rules = cfg.rules ?? [];
  if (rules.length === 0) return { kind: "on", label: cfg.defaultVariant || "On" };

  const rollout = rules.find((r) => r.rollout && r.rollout.length > 0)?.rollout;
  if (rollout) {
    const treatment = rollout
      .filter((b) => b.variant !== cfg.defaultVariant)
      .reduce((sum, b) => sum + b.weight, 0);
    return { kind: "partial", label: `${treatment}%` };
  }
  return { kind: "targeted", label: "Targeted" };
}
