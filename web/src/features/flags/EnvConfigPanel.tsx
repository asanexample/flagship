import { useEffect, useState, type CSSProperties } from "react";
import { useEnvConfig, useSetConfig, type Stage } from "../../lib/queries";
import { relativeTime } from "../../lib/format";
import type { EnvConfig, Flag, ProductRef, Rule } from "../../lib/types";
import { Toggle } from "../../components/ui/Toggle";
import { Field, Select } from "../../components/ui/Field";
import { Spinner } from "../../components/ui/Spinner";
import styles from "./EnvConfigPanel.module.css";

function rolloutWeight(cfg: EnvConfig | null | undefined, variant: string): number {
  const rollout = cfg?.rules?.find((r) => r.rollout && r.rollout.length > 0)?.rollout;
  return rollout?.find((b) => b.variant === variant)?.weight ?? 0;
}

/** One Environment's controls. Every change persists immediately — this is the "instant switch". */
export function EnvConfigPanel({ product, flag, stage }: { product: ProductRef; flag: Flag; stage: Stage }) {
  const variants = Object.keys(flag.variations).sort(); // deterministic, matches the backend default
  const isBool = flag.type === "boolean";

  const { data: cfg, isPending } = useEnvConfig(product.team, product.product, flag.key, stage);
  const save = useSetConfig(product.team, product.product, flag.key, stage);

  const enabled = cfg?.enabled ?? false;
  const defaultVariant = cfg?.defaultVariant || variants[0] || "";
  const treatment = variants.find((v) => v !== defaultVariant) ?? variants[0] ?? "";

  const [pct, setPct] = useState(0);
  useEffect(() => setPct(rolloutWeight(cfg, treatment)), [cfg, treatment]);

  function persist(patch: { enabled?: boolean; defaultVariant?: string; pct?: number }) {
    const nextEnabled = patch.enabled ?? enabled;
    const nextDefault = patch.defaultVariant ?? defaultVariant;
    const nextPct = patch.pct ?? pct;
    const nextTreatment = variants.find((v) => v !== nextDefault) ?? treatment;
    const rules: Rule[] =
      isBool && nextPct > 0 && nextPct < 100
        ? [
            {
              description: `Roll ${nextTreatment} out to ${nextPct}%`,
              rollout: [
                { variant: nextTreatment, weight: nextPct },
                { variant: nextDefault, weight: 100 - nextPct },
              ],
            },
          ]
        : [];
    save.mutate({ enabled: nextEnabled, defaultVariant: nextDefault, rules, updatedBy: "console" });
  }

  if (isPending) {
    return (
      <div className={styles.loading}>
        <Spinner label="Loading configuration" />
      </div>
    );
  }

  return (
    <section className={styles.panel} aria-label={`${stage} configuration`}>
      <div className={styles.killRow}>
        <div>
          <h3 className={styles.title}>
            {stage} <span className={styles.state} data-on={enabled || undefined}>{enabled ? "serving" : "off"}</span>
          </h3>
          <p className={styles.sub}>
            {enabled
              ? "Live in this environment."
              : "Kill switch is off — every request gets the default variant."}
          </p>
        </div>
        <Toggle
          label={`Enable ${flag.key} in ${stage}`}
          checked={enabled}
          onChange={(next) => persist({ enabled: next })}
        />
      </div>

      <div className={styles.controls} data-dim={!enabled || undefined}>
        <Field label="Default variant" hint="Served when no rule matches.">
          <Select
            value={defaultVariant}
            disabled={!enabled}
            onChange={(e) => persist({ defaultVariant: e.target.value })}
          >
            {variants.map((v) => (
              <option key={v} value={v}>
                {v}
              </option>
            ))}
          </Select>
        </Field>

        {isBool ? (
          <div className={styles.rollout}>
            <div className={styles.rolloutHead}>
              <label htmlFor={`rollout-${stage}`}>
                Roll out <span className="mono">{treatment}</span>
              </label>
              <span className={styles.pct}>{pct}%</span>
            </div>
            <input
              id={`rollout-${stage}`}
              className={styles.slider}
              style={{ "--_fill": `${pct}%` } as CSSProperties}
              type="range"
              min={0}
              max={100}
              step={5}
              value={pct}
              disabled={!enabled}
              onChange={(e) => setPct(Number(e.target.value))}
              onPointerUp={() => persist({ pct })}
              onKeyUp={() => persist({ pct })}
            />
            <p className={styles.rolloutHint}>
              {pct === 0
                ? `Everyone gets ${defaultVariant}.`
                : pct === 100
                  ? `Everyone gets ${treatment}.`
                  : `${pct}% get ${treatment}, the rest ${defaultVariant} — sticky per targeting key.`}
            </p>
          </div>
        ) : (
          <p className={styles.note}>A targeting-rule + rollout editor for multivariate flags is coming next.</p>
        )}
      </div>

      <footer className={styles.foot}>
        {save.isPending ? (
          <span className={styles.saving}>
            <Spinner label="Saving" /> Saving…
          </span>
        ) : cfg?.updatedAt && cfg.updatedAt !== "0001-01-01T00:00:00Z" ? (
          <span>
            Updated {relativeTime(cfg.updatedAt)}
            {cfg.updatedBy ? ` by ${cfg.updatedBy}` : ""}
          </span>
        ) : (
          <span className={styles.faint}>Not configured in this environment yet</span>
        )}
        {save.isError && (
          <span className={styles.err} role="alert">
            Couldn’t save — try again.
          </span>
        )}
      </footer>
    </section>
  );
}
