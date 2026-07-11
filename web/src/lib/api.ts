import type { CreateFlagInput, EnvConfig, Flag } from "./types";

/** A typed error carrying the HTTP status and the server's message (from the API's {error} body). */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { "content-type": "application/json", ...init?.headers },
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body?.error) message = body.error;
    } catch {
      /* non-JSON error body — keep the status text */
    }
    throw new ApiError(res.status, message);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

const flagsBase = (team: string, product: string) =>
  `/api/v1/teams/${encodeURIComponent(team)}/products/${encodeURIComponent(product)}/flags`;

const configPath = (team: string, product: string, key: string, stage: string) =>
  `${flagsBase(team, product)}/${encodeURIComponent(key)}/environments/${encodeURIComponent(stage)}`;

export const api = {
  listFlags: (team: string, product: string) => request<Flag[]>(flagsBase(team, product)),

  createFlag: (team: string, product: string, input: CreateFlagInput) =>
    request<Flag>(flagsBase(team, product), { method: "POST", body: JSON.stringify(input) }),

  deleteFlag: (team: string, product: string, key: string) =>
    request<void>(`${flagsBase(team, product)}/${encodeURIComponent(key)}`, { method: "DELETE" }),

  /** Returns null when the flag has no config for this Environment (a 404 is expected, not an error). */
  getConfig: async (team: string, product: string, key: string, stage: string): Promise<EnvConfig | null> => {
    try {
      return await request<EnvConfig>(configPath(team, product, key, stage));
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) return null;
      throw err;
    }
  },

  setConfig: (team: string, product: string, key: string, stage: string, cfg: EnvConfig) =>
    request<EnvConfig>(configPath(team, product, key, stage), { method: "PUT", body: JSON.stringify(cfg) }),
};
