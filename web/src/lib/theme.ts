export type Theme = "light" | "dark";

const KEY = "flagship-theme";

/** Stamp the initial theme on <html> before first paint: saved choice, else system preference. */
export function initTheme(): void {
  const saved = localStorage.getItem(KEY) as Theme | null;
  const theme: Theme = saved ?? (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
  document.documentElement.dataset.theme = theme;
}

export function getTheme(): Theme {
  return (document.documentElement.dataset.theme as Theme) ?? "light";
}

export function toggleTheme(): Theme {
  const next: Theme = getTheme() === "dark" ? "light" : "dark";
  document.documentElement.dataset.theme = next;
  localStorage.setItem(KEY, next);
  return next;
}
