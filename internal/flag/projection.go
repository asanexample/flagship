package flag

import "sort"

// This file is the load-bearing seam of Flagship (ADR-099 D5, "the risky core"): it joins a Flag
// definition with its per-Environment config and renders flagd's flag-set schema. The sync source streams
// exactly this JSON to evaluators, which evaluate it locally. If this projection is correct, we inherit a
// mature, well-tested evaluator for free; if it's wrong, every consumer is wrong — so it is unit-tested
// hard (projection_test.go) rather than trusted.

// FlagSet is flagd's top-level flag-set document: https://flagd.dev/reference/flag-definitions/
type FlagSet struct {
	Flags map[string]FlagdFlag `json:"flags"`
}

// FlagdFlag is one flag in flagd's schema. Targeting is JsonLogic (flagd's dialect, with the custom
// `fractional`, `starts_with`, `ends_with` ops) that returns a variant name; when it returns null or a
// name that isn't a variant, flagd falls back to DefaultVariant — which is how "no rule matched" works.
type FlagdFlag struct {
	State          string         `json:"state"` // ENABLED | DISABLED
	Variants       map[string]any `json:"variants"`
	DefaultVariant string         `json:"defaultVariant"`
	Targeting      any            `json:"targeting,omitempty"`
}

// Project renders one Environment's flags into flagd's schema. It joins each Flag with its EnvConfig
// (keyed by Flag.ID). Semantics:
//   - kill switch: EnvConfig.Enabled=false -> flagd state DISABLED (targeting skipped, DefaultVariant served).
//   - no config for this env, or unknown: DISABLED at a deterministic default (fail safe, never a panic).
//   - otherwise ENABLED with the ordered rules compiled to a single JsonLogic `if` chain.
func Project(flags []Flag, cfgs map[string]EnvConfig) FlagSet {
	out := FlagSet{Flags: make(map[string]FlagdFlag, len(flags))}
	for _, f := range flags {
		fd := FlagdFlag{Variants: f.Variations, State: "DISABLED", DefaultVariant: defaultVariant(f)}
		if cfg, ok := cfgs[f.ID]; ok {
			if cfg.DefaultVariant != "" {
				fd.DefaultVariant = cfg.DefaultVariant
			}
			if cfg.Enabled {
				fd.State = "ENABLED"
				fd.Targeting = compileTargeting(cfg.Rules)
			}
		}
		out.Flags[f.Key] = fd
	}
	return out
}

// defaultVariant is the fallback when a flag has no per-env config: the lexicographically first variant,
// chosen deterministically so an un-configured flag is stable rather than random.
func defaultVariant(f Flag) string {
	names := make([]string, 0, len(f.Variations))
	for n := range f.Variations {
		names = append(names, n)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// compileTargeting turns the ordered rules into `{"if": [cond1, res1, cond2, res2, ...]}`. With an even
// number of args there is no trailing else, so when no condition matches JsonLogic yields null and flagd
// falls back to DefaultVariant. Returns nil (omitted) when there are no rules.
func compileTargeting(rules []Rule) any {
	if len(rules) == 0 {
		return nil
	}
	args := make([]any, 0, len(rules)*2)
	for _, r := range rules {
		args = append(args, compileConditions(r.Conditions), compileResult(r))
	}
	return map[string]any{"if": args}
}

// compileConditions ANDs a rule's conditions. No conditions => always-true (an unconditional rule).
func compileConditions(cs []Condition) any {
	if len(cs) == 0 {
		return true
	}
	terms := make([]any, 0, len(cs))
	for _, c := range cs {
		terms = append(terms, compileCondition(c))
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return map[string]any{"and": terms}
}

func compileCondition(c Condition) any {
	v := map[string]any{"var": c.Attribute}
	switch c.Op {
	case OpEq:
		return map[string]any{"==": []any{v, first(c.Values)}}
	case OpNe:
		return map[string]any{"!=": []any{v, first(c.Values)}}
	case OpIn:
		return map[string]any{"in": []any{v, toAnySlice(c.Values)}}
	case OpNotIn:
		return map[string]any{"!": map[string]any{"in": []any{v, toAnySlice(c.Values)}}}
	case OpStartsWith:
		return map[string]any{"starts_with": []any{v, first(c.Values)}} // flagd custom op
	case OpEndsWith:
		return map[string]any{"ends_with": []any{v, first(c.Values)}} // flagd custom op
	default:
		// Unknown operator: emit a never-match so a bad rule fails closed to DefaultVariant rather than
		// matching everything. (The management API validates operators before persist; this is defence.)
		return false
	}
}

// compileResult is either a fixed variant name, or flagd's `fractional` op keyed on the caller's
// targetingKey for a sticky percentage split: {"fractional": [{"var":"targetingKey"}, ["v1", w1], ...]}.
func compileResult(r Rule) any {
	if len(r.Rollout) == 0 {
		return r.Variant
	}
	buckets := make([]any, 0, len(r.Rollout)+1)
	buckets = append(buckets, map[string]any{"var": "targetingKey"})
	for _, b := range r.Rollout {
		buckets = append(buckets, []any{b.Variant, b.Weight})
	}
	return map[string]any{"fractional": buckets}
}

func first(vs []string) any {
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}

func toAnySlice(vs []string) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = v
	}
	return out
}
