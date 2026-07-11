import type { Operator, Rule } from "../../lib/types";
import styles from "./RuleBuilder.module.css";

const OPERATORS: { value: Operator; label: string }[] = [
  { value: "eq", label: "is" },
  { value: "ne", label: "is not" },
  { value: "in", label: "is any of" },
  { value: "not_in", label: "is none of" },
  { value: "starts_with", label: "starts with" },
  { value: "ends_with", label: "ends with" },
];

const MULTI_VALUE: Operator[] = ["in", "not_in"];

/**
 * Edits the ordered targeting rules (conditions → variant). First match wins; anything unmatched falls to
 * the rollout / default. Produces our friendly Rule model — the backend compiles it to flagd JsonLogic.
 */
export function RuleBuilder({
  rules,
  variants,
  onChange,
}: {
  rules: Rule[];
  variants: string[];
  onChange: (rules: Rule[]) => void;
}) {
  const update = (i: number, rule: Rule) => onChange(rules.map((r, j) => (j === i ? rule : r)));
  const remove = (i: number) => onChange(rules.filter((_, j) => j !== i));
  const add = () =>
    onChange([...rules, { conditions: [{ attribute: "", op: "eq", values: [""] }], variant: variants[0] ?? "" }]);

  return (
    <div className={styles.builder}>
      {rules.length === 0 && (
        <p className={styles.empty}>No targeting rules — everyone gets the rollout, then the default.</p>
      )}
      {rules.map((rule, i) => (
        <RuleCard
          key={i}
          index={i}
          rule={rule}
          variants={variants}
          onChange={(r) => update(i, r)}
          onRemove={() => remove(i)}
        />
      ))}
      <button type="button" className={styles.addRule} onClick={add}>
        + Add rule
      </button>
    </div>
  );
}

function RuleCard({
  index,
  rule,
  variants,
  onChange,
  onRemove,
}: {
  index: number;
  rule: Rule;
  variants: string[];
  onChange: (rule: Rule) => void;
  onRemove: () => void;
}) {
  const conditions = rule.conditions ?? [];

  const setCondition = (ci: number, patch: Partial<{ attribute: string; op: Operator; values: string[] }>) =>
    onChange({
      ...rule,
      conditions: conditions.map((c, j) => (j === ci ? { ...c, ...patch } : c)),
    });
  const addCondition = () =>
    onChange({ ...rule, conditions: [...conditions, { attribute: "", op: "eq", values: [""] }] });
  const removeCondition = (ci: number) =>
    onChange({ ...rule, conditions: conditions.filter((_, j) => j !== ci) });

  return (
    <div className={styles.card}>
      <div className={styles.cardHead}>
        <span className={styles.ordinal}>Rule {index + 1}</span>
        <button type="button" className={styles.iconBtn} onClick={onRemove} aria-label={`Remove rule ${index + 1}`}>
          ✕
        </button>
      </div>

      <div className={styles.conditions}>
        {conditions.map((c, ci) => (
          <div key={ci} className={styles.condition}>
            <span className={styles.joiner}>{ci === 0 ? "If" : "and"}</span>
            <input
              className={`${styles.attr} mono`}
              placeholder="attribute"
              aria-label="Attribute"
              value={c.attribute}
              onChange={(e) => setCondition(ci, { attribute: e.target.value })}
            />
            <select
              className={styles.op}
              aria-label="Operator"
              value={c.op}
              onChange={(e) => setCondition(ci, { op: e.target.value as Operator })}
            >
              {OPERATORS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
            <input
              className={styles.values}
              placeholder={MULTI_VALUE.includes(c.op) ? "acme, globex" : "value"}
              aria-label="Value"
              // Hold the raw text while typing (join without trimming) so commas survive; the values are
              // trimmed + de-blanked at save time in composeRules.
              value={c.values.join(",")}
              onChange={(e) => setCondition(ci, { values: e.target.value.split(",") })}
            />
            {conditions.length > 1 && (
              <button
                type="button"
                className={styles.iconBtn}
                onClick={() => removeCondition(ci)}
                aria-label="Remove condition"
              >
                ✕
              </button>
            )}
          </div>
        ))}
        <button type="button" className={styles.addCond} onClick={addCondition}>
          + and
        </button>
      </div>

      <div className={styles.serve}>
        <span className={styles.joiner}>serve</span>
        <select
          className={styles.variant}
          aria-label="Variant to serve"
          value={rule.variant ?? variants[0] ?? ""}
          onChange={(e) => onChange({ ...rule, variant: e.target.value, rollout: undefined })}
        >
          {variants.map((v) => (
            <option key={v} value={v}>
              {v}
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}
