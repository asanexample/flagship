import { useEffect, useRef, useState, type CSSProperties } from "react";
import { useEnvConfig, useSetConfig, type Stage } from "../../lib/queries";
import { relativeTime } from "../../lib/format";
import type { EnvConfig, Flag, ProductRef, Rule } from "../../lib/types";
import { Toggle } from "../../components/ui/Toggle";
import { Button } from "../../components/ui/Button";
import { Field, Select } from "../../components/ui/Field";
import { Spinner } from "../../components/ui/Spinner";
import { RuleBuilder } from "./RuleBuilder";
import styles from "./EnvConfigPanel.module.css";

/** The editable form: conditional targeting rules and an unconditional rollout are modelled separately,
 *  then composed back into one ordered rule list on save. */
interface Draft {
  enabled: boolean;
  defaultVariant: string;
  rules: Rule[]; // conditional targeting rules only
  rolloutPct: number;
}

const isRollout = (r: Rule) => (r.conditions?.length ?? 0) === 0 && (r.rollout?.length ?? 0) > 0;

function seedDraft(cfg: EnvConfig | null | undefined, variants: string[]): Draft {
  const defaultVariant = cfg?.defaultVariant || variants[0] || "";
  const all = cfg?.rules ?? [];
  const treatment = variants.find((v) => v !== defaultVariant) ?? "";
  const rolloutPct = all.find(isRollout)?.rollout?.find((b) => b.variant === treatment)?.weight ?? 0;
  return {
    enabled: cfg?.enabled ?? false,
    defaultVariant,
    rules: all.filter((r) => (r.conditions?.length ?? 0) > 0),
    rolloutPct,
  };
}

function composeRules(draft: Draft, variants: string[], isBool: boolean): Rule[] {
  // Sanitize the targeting rules at save time: trim/de-blank values, drop empty conditions and any rule
  // left without a usable condition or a result variant.
  const conditional = draft.rules
    .map((r) => ({
      ...r,
      conditions: (r.conditions ?? [])
        .map((c) => ({ ...c, values: c.values.map((v) => v.trim()).filter(Boolean) }))
        .filter((c) => c.attribute.trim() !== "" && c.values.length > 0),
    }))
    .filter((r) => (r.conditions?.length ?? 0) > 0 && r.variant);
  const treatment = variants.find((v) => v !== draft.defaultVariant) ?? "";
  const rollout: Rule[] =
    isBool && draft.rolloutPct > 0 && draft.rolloutPct < 100 && treatment
      ? [
          {
            description: `Roll ${treatment} out to ${draft.rolloutPct}%`,
            rollout: [
              { variant: treatment, weight: draft.rolloutPct },
              { variant: draft.defaultVariant, weight: 100 - draft.rolloutPct },
            ],
          },
        ]
      : [];
  return [...conditional, ...rollout];
}

export function EnvConfigPanel({ product, flag, stage }: { product: ProductRef; flag: Flag; stage: Stage }) {
  const variants = Object.keys(flag.variations).sort();
  const isBool = flag.type === "boolean";
  const { data: cfg, isPending } = useEnvConfig(product.team, product.product, flag.key, stage);
  const save = useSetConfig(product.team, product.product, flag.key, stage);

  const [draft, setDraft] = useState<Draft>(() => seedDraft(cfg, variants));
  const [touched, setTouched] = useState(false);
  const touchedRef = useRef(false);
  const mark = (t: boolean) => {
    touchedRef.current = t;
    setTouched(t);
  };

  // Re-seed from the server whenever it changes — but only while the user has no unsaved edits.
  useEffect(() => {
    if (!touchedRef.current) setDraft(seedDraft(cfg, variants));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cfg]);

  const edit = (patch: Partial<Draft>) => {
    mark(true);
    setDraft((d) => ({ ...d, ...patch }));
  };

  const persist = (next: Draft) => {
    mark(false);
    save.mutate({
      enabled: next.enabled,
      defaultVariant: next.defaultVariant,
      rules: composeRules(next, variants, isBool),
      updatedBy: "console",
    });
  };

  // The kill switch is the instant control: flip and commit (including any pending edits).
  const toggleKill = (enabled: boolean) => {
    const next = { ...draft, enabled };
    setDraft(next);
    persist(next);
  };

  const discard = () => {
    mark(false);
    setDraft(seedDraft(cfg, variants));
  };

  if (isPending) {
    return (
      <div className={styles.loading}>
        <Spinner label="Loading configuration" />
      </div>
    );
  }

  const treatment = variants.find((v) => v !== draft.defaultVariant) ?? variants[0] ?? "";

  return (
    <section className={styles.panel} aria-label={`${stage} configuration`}>
      <div className={styles.killRow}>
        <div>
          <h3 className={styles.title}>
            {stage} <span className={styles.state} data-on={draft.enabled || undefined}>{draft.enabled ? "serving" : "off"}</span>
          </h3>
          <p className={styles.sub}>
            {draft.enabled
              ? "Live in this environment."
              : "Kill switch is off — every request gets the default variant."}
          </p>
        </div>
        <Toggle label={`Enable ${flag.key} in ${stage}`} checked={draft.enabled} onChange={toggleKill} />
      </div>

      <div className={styles.controls} data-dim={!draft.enabled || undefined}>
        <Field label="Default variant" hint="Served when no rule or rollout matches.">
          <Select
            value={draft.defaultVariant}
            disabled={!draft.enabled}
            onChange={(e) => edit({ defaultVariant: e.target.value })}
          >
            {variants.map((v) => (
              <option key={v} value={v}>
                {v}
              </option>
            ))}
          </Select>
        </Field>

        <div className={styles.group}>
          <span className={styles.groupLabel}>Targeting rules</span>
          <RuleBuilder rules={draft.rules} variants={variants} onChange={(rules) => edit({ rules })} />
        </div>

        {isBool && (
          <div className={styles.rollout}>
            <div className={styles.rolloutHead}>
              <label htmlFor={`rollout-${stage}`}>
                Roll out <span className="mono">{treatment}</span>
              </label>
              <span className={styles.pct}>{draft.rolloutPct}%</span>
            </div>
            <input
              id={`rollout-${stage}`}
              className={styles.slider}
              style={{ "--_fill": `${draft.rolloutPct}%` } as CSSProperties}
              type="range"
              min={0}
              max={100}
              step={5}
              value={draft.rolloutPct}
              disabled={!draft.enabled}
              onChange={(e) => edit({ rolloutPct: Number(e.target.value) })}
            />
            <p className={styles.rolloutHint}>
              {draft.rolloutPct === 0
                ? "No gradual rollout."
                : draft.rolloutPct === 100
                  ? `Everyone past the rules gets ${treatment}.`
                  : `${draft.rolloutPct}% of the rest get ${treatment} — sticky per targeting key.`}
            </p>
          </div>
        )}
      </div>

      <footer className={styles.foot}>
        {save.isPending ? (
          <span className={styles.saving}>
            <Spinner label="Saving" /> Saving…
          </span>
        ) : touched ? (
          <div className={styles.saveRow}>
            <span className={styles.unsaved}>Unsaved changes</span>
            <div className={styles.saveActions}>
              <Button variant="ghost" size="sm" onClick={discard}>
                Discard
              </Button>
              <Button variant="primary" size="sm" onClick={() => persist(draft)}>
                Save changes
              </Button>
            </div>
          </div>
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
