// Domain types — the wire shapes of Flagship's management API (Go internal/flag + internal/api).
// The frontend works entirely in these friendly shapes; the backend projects them to flagd JsonLogic.

export type FlagType = "boolean" | "string" | "number" | "json";

export type Operator = "eq" | "ne" | "in" | "not_in" | "starts_with" | "ends_with";

export interface ProductRef {
  team: string;
  product: string;
}

export interface EnvRef {
  team: string;
  product: string;
  stage: string;
}

export interface Flag {
  id: string;
  product: ProductRef;
  key: string;
  description: string;
  type: FlagType;
  variations: Record<string, unknown>;
  createdAt: string;
}

export interface Condition {
  attribute: string;
  op: Operator;
  values: string[];
}

export interface RolloutBucket {
  variant: string;
  weight: number;
}

/** A targeting rule: conditions (AND) resolve to either a fixed variant or a percentage rollout. */
export interface Rule {
  description?: string;
  conditions?: Condition[];
  variant?: string;
  rollout?: RolloutBucket[];
}

/** A flag's behaviour in one Environment. */
export interface EnvConfig {
  flagId?: string;
  env?: EnvRef;
  enabled: boolean;
  defaultVariant: string;
  rules: Rule[];
  updatedBy?: string;
  updatedAt?: string;
}

export interface CreateFlagInput {
  key: string;
  description: string;
  type: FlagType;
  variations: Record<string, unknown>;
}
