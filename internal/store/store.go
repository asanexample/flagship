// Package store is the persistence seam for Flagship's control plane (ADR-099 D2). The management API
// writes through it; the sync source reads an Environment's flags + configs through it and hands them to
// flag.Project. A Postgres (CNPG) implementation lives alongside; unit tests use an in-memory fake.
package store

import (
	"context"

	"github.com/asanexample/flagship/internal/flag"
)

// Store persists flag definitions and their per-Environment config, and serves the sync source's read
// path. Implementations must be safe for concurrent use.
type Store interface {
	// --- Flag definitions (Product-scoped) ---

	CreateFlag(ctx context.Context, f flag.Flag) error
	GetFlag(ctx context.Context, product flag.ProductRef, key string) (flag.Flag, error)
	ListFlags(ctx context.Context, product flag.ProductRef) ([]flag.Flag, error)
	DeleteFlag(ctx context.Context, product flag.ProductRef, key string) error

	// --- Per-Environment config (the object a dashboard toggle edits) ---

	GetEnvConfig(ctx context.Context, flagID string, env flag.EnvRef) (flag.EnvConfig, error)
	// SetEnvConfig upserts the config AND appends an audit record atomically (ADR-099 D2): a change and
	// its audit trail must never diverge.
	SetEnvConfig(ctx context.Context, cfg flag.EnvConfig) error

	// --- Sync-source read path ---

	// EnvFlagSet returns every flag in an Environment's Product plus that flag's config for the
	// Environment — the exact input to flag.Project. This is the hot read behind the sync stream, so
	// implementations should keep it a single round trip.
	EnvFlagSet(ctx context.Context, env flag.EnvRef) (flags []flag.Flag, cfgs map[string]flag.EnvConfig, err error)
}

// ErrNotFound is returned by Get* when the requested flag or config does not exist.
type ErrNotFound struct{ What string }

func (e ErrNotFound) Error() string { return e.What + " not found" }
