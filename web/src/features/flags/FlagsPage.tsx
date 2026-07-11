import { useState } from "react";
import { Link } from "react-router-dom";
import { useFlags } from "../../lib/queries";
import { relativeTime } from "../../lib/format";
import type { ProductRef } from "../../lib/types";
import { Button } from "../../components/ui/Button";
import { Badge } from "../../components/ui/Badge";
import { Spinner } from "../../components/ui/Spinner";
import { EmptyState } from "../../components/ui/EmptyState";
import { CreateFlagDialog } from "./CreateFlagDialog";
import styles from "./FlagsPage.module.css";

export function FlagsPage({ product }: { product: ProductRef }) {
  const { data: flags, isPending, isError, error } = useFlags(product.team, product.product);
  const [creating, setCreating] = useState(false);

  return (
    <div>
      <header className={styles.header}>
        <div>
          <h1 className={styles.title}>Flags</h1>
          <p className={styles.subtitle}>
            Runtime feature flags for <span className="mono">{product.team}/{product.product}</span>.
          </p>
        </div>
        <Button variant="primary" onClick={() => setCreating(true)}>
          New flag
        </Button>
      </header>

      {isPending && (
        <div className={styles.center}>
          <Spinner label="Loading flags" />
        </div>
      )}

      {isError && (
        <div className={styles.error} role="alert">
          Couldn’t load flags — {error instanceof Error ? error.message : "unknown error"}.
        </div>
      )}

      {flags && flags.length === 0 && (
        <EmptyState
          title="No flags yet"
          description="Create your first flag to gate a feature behind a kill switch or roll it out gradually."
          action={
            <Button variant="primary" onClick={() => setCreating(true)}>
              New flag
            </Button>
          }
        />
      )}

      {flags && flags.length > 0 && (
        <div className={styles.table}>
          <div className={styles.thead}>
            <span>Key</span>
            <span>Type</span>
            <span>Description</span>
            <span>Created</span>
          </div>
          {flags.map((flag) => (
            <Link key={flag.id} to={`/flags/${encodeURIComponent(flag.key)}`} className={styles.row}>
              <span className={`${styles.key} mono`}>{flag.key}</span>
              <span>
                <Badge tone="neutral">{flag.type}</Badge>
              </span>
              <span className={styles.desc}>
                {flag.description || <span className={styles.faint}>No description</span>}
              </span>
              <span className={styles.created}>{relativeTime(flag.createdAt)}</span>
            </Link>
          ))}
        </div>
      )}

      <CreateFlagDialog product={product} open={creating} onClose={() => setCreating(false)} />
    </div>
  );
}
