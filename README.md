# Flagship

**Flagship is the platform's feature-flag service** — a first-class platform primitive for
release-gating, kill switches, tenant entitlements, and (later) experimentation. It is the
implementation of **[ADR-099](https://github.com/asanexample/platform/blob/main/docs/adrs/099-feature-flags-platform-service.md)**.

> Working name; trivially renamed.

## The one idea

**We do not evaluate flags. flagd does.**

Flagship owns the **control plane** (define flags, targeting, per-environment config; store; audit;
RBAC) and a **sync source** that *projects* our model into [flagd](https://flagd.dev)'s flag-set JSON.
Consuming apps evaluate **locally** via the standard **[OpenFeature](https://openfeature.dev) SDK + flagd
provider** — sub-millisecond, no per-eval network call, and fail-static if we're briefly down. We build
the part OSS does poorly (the product); we reuse the proven evaluator and SDKs the way we reuse HTTP.

```
                          Flagship (hub)                         workload cluster
  ┌───────────────┐   ┌──────────────────────┐            ┌───────────────────────────┐
  │  Dashboard    │──▶│  Management API       │            │  app (Go/TS/Py)           │
  │  (Next.js)    │   │  ├─ Postgres (CNPG)   │            │  ├─ OpenFeature SDK       │
  └───────────────┘   │  ├─ Audit             │  flagd     │  └─ flagd provider ◀──┐   │
                      │  └─ Sync source ──────┼──sync──────┼──▶ (local eval)       │   │
                      └──────────────────────┘  (scoped,   │   sync from local relay┘   │
                                                 per-Env)  └───────────────────────────┘
```

(The per-cluster **sync relay** between the hub sync source and app evaluators — so we don't stream the
hub to every pod, and so each consumer only ever sees its own Environment's flags — is [ADR-099](https://github.com/asanexample/platform/blob/main/docs/adrs/099-feature-flags-platform-service.md)
D5 and Phase 3 below.)

## Domain model (`internal/flag`)

- **`Flag`** — a Product-scoped *definition*: key, type (`boolean|string|number|json`), and its
  `variations` (name → value). Defined once, across all stages.
- **`EnvConfig`** — a flag's behaviour in *one* Environment: the `Enabled` kill switch, a
  `DefaultVariant`, and ordered `Rules`. This is what a dashboard toggle edits and what propagates in
  seconds.
- **`Rule`** — `Conditions` (attribute comparisons, AND'd) → either a fixed `Variant` or a `Rollout`
  (deterministic percentage split, sticky by the caller's `targetingKey`).
- **`Project(flags, cfgs)`** (`projection.go`) — the load-bearing seam: joins definitions with
  per-Environment config and renders flagd's schema (kill switch → `state: DISABLED`; rules → a JsonLogic
  `if`-chain; rollout → flagd's `fractional` op). Heavily unit-tested (`projection_test.go`) — a bug here
  poisons every consumer.

Tenancy is the platform's existing `Team → Product → Environment` ([ADR-067]); no new concept.

## Layout

```
cmd/flagship/        service entrypoint (API + sync source)
internal/flag/       domain model + projection to flagd  ← the core, done + tested
internal/store/      persistence seam (Store interface); Postgres impl
internal/syncsource/ flagd-compatible sync stream (per-Environment, authenticated)
migrations/          Postgres schema (CNPG)
```

## Build phases (this is a multi-session build — see the ADR)

1. **Foundation + spine** ← *in progress.* Domain model + projection (done, tested), store interface,
   schema. Next: Postgres store impl, the flagd sync source, and wire **alpha-shop** as the first
   consumer (a flag-gated checkout variant that shows up as a `feature_flag.variant` span attribute).
2. **Product.** Management API + Next.js dashboard + Keycloak RBAC + audit view.
3. **Delivery at scale.** Per-cluster sync relay, sidecar-vs-in-process decision, and the platform
   Product / GitOps wiring on the [ADR-081] delivery road.

## Develop

```bash
go build ./...
go test ./...
```

[ADR-067]: https://github.com/asanexample/platform/blob/main/docs/adrs/067-idp-domain-model.md
[ADR-081]: https://github.com/asanexample/platform/blob/main/docs/adrs/081-platform-service-delivery.md
