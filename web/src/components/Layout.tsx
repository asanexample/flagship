import { useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { getTheme, toggleTheme } from "../lib/theme";
import type { ProductRef } from "../lib/types";
import styles from "./Layout.module.css";

export function Layout({ product, children }: { product: ProductRef; children: ReactNode }) {
  const [theme, setTheme] = useState(getTheme());

  return (
    <div className={styles.app}>
      <header className={styles.bar}>
        <Link to="/" className={styles.brand}>
          <span className={styles.mark} aria-hidden>
            ⚑
          </span>
          <span className={styles.word}>Flagship</span>
        </Link>

        <div className={styles.context} title="The Product these flags belong to">
          <span className={styles.ctxLabel}>Product</span>
          <span className={`${styles.ctxValue} mono`}>
            {product.team}/{product.product}
          </span>
        </div>

        <button
          type="button"
          className={styles.theme}
          onClick={() => setTheme(toggleTheme())}
          aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}
        >
          {theme === "dark" ? "☾" : "☀"}
        </button>
      </header>

      <main className={styles.main}>{children}</main>
    </div>
  );
}
