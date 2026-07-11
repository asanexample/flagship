import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./api";
import type { CreateFlagInput, EnvConfig } from "./types";

/** The platform's Environment stages (ADR-067), in pipeline order. */
export const STAGES = ["dev", "test", "uat", "staging", "prod"] as const;
export type Stage = (typeof STAGES)[number];

const queryKeys = {
  flags: (team: string, product: string) => ["flags", team, product] as const,
  config: (team: string, product: string, key: string, stage: string) =>
    ["config", team, product, key, stage] as const,
};

export function useFlags(team: string, product: string) {
  return useQuery({
    queryKey: queryKeys.flags(team, product),
    queryFn: () => api.listFlags(team, product),
  });
}

export function useCreateFlag(team: string, product: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateFlagInput) => api.createFlag(team, product, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.flags(team, product) }),
  });
}

export function useDeleteFlag(team: string, product: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => api.deleteFlag(team, product, key),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.flags(team, product) }),
  });
}

export function useEnvConfig(team: string, product: string, key: string, stage: string) {
  return useQuery({
    queryKey: queryKeys.config(team, product, key, stage),
    queryFn: () => api.getConfig(team, product, key, stage),
  });
}

export function useSetConfig(team: string, product: string, key: string, stage: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cfg: EnvConfig) => api.setConfig(team, product, key, stage, cfg),
    // Write the fresh config straight into the cache so the UI reflects the change immediately.
    onSuccess: (data) => qc.setQueryData(queryKeys.config(team, product, key, stage), data),
  });
}
