import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useFlags, type Stage } from "../../lib/queries";
import type { ProductRef } from "../../lib/types";
import { Badge } from "../../components/ui/Badge";
import { Spinner } from "../../components/ui/Spinner";
import { EnvStrip } from "./EnvStrip";
import { EnvConfigPanel } from "./EnvConfigPanel";
import styles from "./FlagDetail.module.css";

export function FlagDetail({ product }: { product: ProductRef }) {
  const { key = "" } = useParams();
  const { data: flags, isPending } = useFlags(product.team, product.product);
  const flag = flags?.find((f) => f.key === key);
  const [stage, setStage] = useState<Stage>("prod");

  if (isPending) {
    return (
      <div className={styles.center}>
        <Spinner label="Loading flag" />
      </div>
    );
  }

  if (!flag) {
    return (
      <div className={styles.missing}>
        Flag <span className="mono">{key}</span> not found. <Link to="/">Back to flags</Link>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <Link to="/" className={styles.back}>
        ← Flags
      </Link>

      <header className={styles.head}>
        <h1 className={`${styles.key} mono`}>{flag.key}</h1>
        <Badge tone="neutral">{flag.type}</Badge>
      </header>
      {flag.description && <p className={styles.desc}>{flag.description}</p>}

      <div className={styles.variations}>
        {Object.keys(flag.variations)
          .sort()
          .map((name) => (
            <span key={name} className={styles.variant}>
              <span className="mono">{name}</span>
              <code className={styles.val}>{JSON.stringify(flag.variations[name])}</code>
            </span>
          ))}
      </div>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Environments</h2>
        <EnvStrip product={product} flagKey={flag.key} selected={stage} onSelect={setStage} />
      </section>

      <EnvConfigPanel key={stage} product={product} flag={flag} stage={stage} />
    </div>
  );
}
