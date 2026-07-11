import { STAGES, useEnvConfig, type Stage } from "../../lib/queries";
import { summarize } from "../../lib/format";
import type { ProductRef } from "../../lib/types";
import styles from "./EnvStrip.module.css";

/**
 * The signature element: a flag's state across the dev → prod pipeline, one cell per stage, so exposure is
 * readable at a glance. Each cell fetches its own stage's config (cached + shared with the config panel).
 */
export function EnvStrip({
  product,
  flagKey,
  selected,
  onSelect,
}: {
  product: ProductRef;
  flagKey: string;
  selected?: Stage;
  onSelect?: (stage: Stage) => void;
}) {
  return (
    <div className={styles.strip} role="tablist" aria-label="Environments">
      {STAGES.map((stage) => (
        <StageCell
          key={stage}
          product={product}
          flagKey={flagKey}
          stage={stage}
          selected={selected === stage}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}

function StageCell({
  product,
  flagKey,
  stage,
  selected,
  onSelect,
}: {
  product: ProductRef;
  flagKey: string;
  stage: Stage;
  selected: boolean;
  onSelect?: (stage: Stage) => void;
}) {
  const { data, isPending } = useEnvConfig(product.team, product.product, flagKey, stage);
  const state = summarize(data);
  const interactive = Boolean(onSelect);

  return (
    <button
      type="button"
      role={interactive ? "tab" : undefined}
      aria-selected={interactive ? selected : undefined}
      tabIndex={interactive ? 0 : -1}
      disabled={!interactive}
      className={styles.cell}
      data-state={state.kind}
      data-selected={selected || undefined}
      data-prod={stage === "prod" || undefined}
      onClick={() => onSelect?.(stage)}
    >
      <span className={styles.stage}>{stage}</span>
      <span className={styles.value}>{isPending ? "·" : state.label}</span>
    </button>
  );
}
