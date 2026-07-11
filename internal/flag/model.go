// Package flag is the domain model for Flagship, the platform feature-flag service (ADR-099).
//
// Design note — we do NOT evaluate flags here. Evaluation happens client-side in flagd (via the
// OpenFeature SDK). Flagship owns the *definition* of flags and *projects* them into flagd's flag-set
// JSON, which the sync source streams to evaluators. So this package models "what a flag is and how it's
// configured per environment", and package projection turns that into flagd's wire schema. Keeping the
// evaluator upstream is what keeps this service small.
package flag

import "time"

// Type is the value type a flag's variations carry. It mirrors the shapes flagd/OpenFeature support.
type Type string

const (
	Bool   Type = "boolean"
	String Type = "string"
	Number Type = "number"
	JSON   Type = "json"
)

// ProductRef identifies the owning tenant (Team + Product) — the tenancy axis from ADR-067.
type ProductRef struct {
	Team    string `json:"team"`
	Product string `json:"product"`
}

// EnvRef identifies a single platform Environment (a Product at a Stage, ADR-067). A flag's behaviour is
// configured per EnvRef, so the same flag can be off in dev and a 5% rollout in prod.
type EnvRef struct {
	Team    string `json:"team"`
	Product string `json:"product"`
	Stage   string `json:"stage"` // dev | test | uat | staging | prod
}

// Flag is a Product-scoped flag DEFINITION: its identity, value type, and the set of variations it can
// resolve to. It carries no per-environment behaviour — that lives in EnvConfig. Defining the variations
// once (here) and only their selection per-environment keeps a flag coherent across stages.
type Flag struct {
	ID          string         `json:"id"`
	Product     ProductRef     `json:"product"`
	Key         string         `json:"key"` // stable evaluation key, unique within a Product (what code asks for)
	Description string         `json:"description"`
	Type        Type           `json:"type"`
	Variations  map[string]any `json:"variations"` // variant name -> value; every value must match Type
	CreatedAt   time.Time      `json:"createdAt"`
}

// EnvConfig is a Flag's behaviour in ONE Environment. This is the object a dashboard toggle edits and
// what propagates to evaluators within seconds (ADR-099 D3). It is deliberately small:
//   - Enabled is the top-level kill switch: false => everyone gets DefaultVariant, rules ignored.
//   - Rules are evaluated in order; the first whose Conditions all match wins.
//   - DefaultVariant is served when no rule matches (and whenever Enabled is false).
type EnvConfig struct {
	FlagID         string `json:"flagId"`
	Env            EnvRef `json:"env"`
	Enabled        bool   `json:"enabled"`
	DefaultVariant string `json:"defaultVariant"`
	Rules          []Rule `json:"rules"`

	// Audit — every change records who and when (ADR-099 D2).
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
}

// Rule maps a matched evaluation context to a result. Exactly one of Variant or Rollout is set: a fixed
// variant, or a deterministic percentage split.
type Rule struct {
	Description string      `json:"description,omitempty"`
	Conditions  []Condition `json:"conditions,omitempty"` // ALL must hold for the rule to match (implicit AND)

	// Result — set exactly one:
	Variant string          `json:"variant,omitempty"` // a fixed variant, OR
	Rollout []RolloutBucket `json:"rollout,omitempty"` // a percentage split bucketed by the caller's targetingKey
}

// Condition is a single comparison over an attribute of the evaluation context (e.g. tenant, email,
// country). v1 supports a small, closed operator set; it compiles to flagd JsonLogic on projection.
type Condition struct {
	Attribute string   `json:"attribute"`
	Op        Operator `json:"op"`
	Values    []string `json:"values"`
}

// Operator is the closed set of comparisons v1 supports. New operators are additive.
type Operator string

const (
	OpEq         Operator = "eq"          // attribute == Values[0]
	OpNe         Operator = "ne"          // attribute != Values[0]
	OpIn         Operator = "in"          // attribute ∈ Values
	OpNotIn      Operator = "not_in"      // attribute ∉ Values
	OpStartsWith Operator = "starts_with" // attribute has prefix Values[0]
	OpEndsWith   Operator = "ends_with"   // attribute has suffix Values[0]
)

// RolloutBucket is one slice of a percentage rollout. Weights across a Rule's buckets must sum to 100.
// A context is placed by a stable hash of its targetingKey (flagd's `fractional` op), so a given subject
// always lands in the same bucket — sticky across evaluations (ADR-099 D7). Callers MUST supply a stable
// targetingKey (user/session/tenant) or a rollout will flap per request.
type RolloutBucket struct {
	Variant string `json:"variant"`
	Weight  int    `json:"weight"` // 0..100
}
