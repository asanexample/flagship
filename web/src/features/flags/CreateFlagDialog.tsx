import { useState, type FormEvent } from "react";
import { ApiError } from "../../lib/api";
import { useCreateFlag } from "../../lib/queries";
import type { ProductRef } from "../../lib/types";
import { Dialog } from "../../components/ui/Dialog";
import { Field, Input } from "../../components/ui/Field";
import { Button } from "../../components/ui/Button";
import styles from "./CreateFlagDialog.module.css";

const KEY_RE = /^[a-z0-9][a-z0-9-]*$/;

export function CreateFlagDialog({
  product,
  open,
  onClose,
}: {
  product: ProductRef;
  open: boolean;
  onClose: () => void;
}) {
  const [key, setKey] = useState("");
  const [description, setDescription] = useState("");
  const create = useCreateFlag(product.team, product.product);

  const keyValid = KEY_RE.test(key);

  function close() {
    setKey("");
    setDescription("");
    create.reset();
    onClose();
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!keyValid) return;
    // v1 creates boolean flags (on/off); multivariate + a variants editor is the next iteration.
    create.mutate(
      { key, description, type: "boolean", variations: { on: true, off: false } },
      { onSuccess: close },
    );
  }

  return (
    <Dialog open={open} onClose={close} title="New flag">
      <form onSubmit={submit} className={styles.form}>
        <Field
          label="Key"
          htmlFor="flag-key"
          hint="Lowercase letters, digits and dashes — this is what your code asks for."
        >
          <Input
            id="flag-key"
            className="mono"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="checkout-variant"
            autoFocus
            autoComplete="off"
            spellCheck={false}
            aria-invalid={key.length > 0 && !keyValid}
          />
        </Field>
        {key.length > 0 && !keyValid && (
          <p className={styles.invalid}>Use lowercase letters, digits and dashes, e.g. checkout-variant.</p>
        )}

        <Field label="Description" htmlFor="flag-desc">
          <Input
            id="flag-desc"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="What does this flag control?"
          />
        </Field>

        <div className={styles.type}>
          <span className={styles.typeLabel}>Type</span>
          <strong>Boolean</strong>
          <span className={styles.variants}>
            <code>on</code> <code>off</code>
          </span>
        </div>

        {create.isError && (
          <p className={styles.error} role="alert">
            {create.error instanceof ApiError ? create.error.message : "Couldn’t create the flag."}
          </p>
        )}

        <div className={styles.actions}>
          <Button type="button" variant="ghost" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={create.isPending} disabled={!keyValid}>
            Create flag
          </Button>
        </div>
      </form>
    </Dialog>
  );
}
