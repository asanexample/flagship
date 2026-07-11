package store

import (
	"context"
	"os"
	"testing"

	"github.com/asanexample/flagship/internal/flag"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPostgres_RoundTrip exercises the Postgres store against a real database. It skips unless
// FLAGSHIP_TEST_DSN points at a throwaway Postgres, and (re)applies migrations/0001_init.sql itself so it
// is self-contained. Run it with: FLAGSHIP_TEST_DSN=postgres://... go test ./internal/store/ -run Postgres
func TestPostgres_RoundTrip(t *testing.T) {
	dsn := os.Getenv("FLAGSHIP_TEST_DSN")
	if dsn == "" {
		t.Skip("set FLAGSHIP_TEST_DSN to run the Postgres store test")
	}
	ctx := context.Background()

	// Fresh schema.
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	schema, err := os.ReadFile("../../migrations/0001_init.sql")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if _, err := pool.Exec(ctx, "DROP TABLE IF EXISTS audit_log, env_configs, flags CASCADE"); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if _, err := pool.Exec(ctx, string(schema)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool.Close()

	st, err := NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	defer st.Close()

	shop := flag.ProductRef{Team: "alpha", Product: "shop"}

	// create
	if err := st.CreateFlag(ctx, flag.Flag{
		Product: shop, Key: "checkout-variant", Type: flag.Bool,
		Variations: map[string]any{"on": true, "off": false},
	}); err != nil {
		t.Fatalf("CreateFlag: %v", err)
	}
	// duplicate -> conflict
	if err := st.CreateFlag(ctx, flag.Flag{Product: shop, Key: "checkout-variant", Type: flag.Bool,
		Variations: map[string]any{"on": true}}); err == nil {
		t.Fatal("duplicate CreateFlag should conflict")
	} else if _, ok := err.(ErrConflict); !ok {
		t.Fatalf("want ErrConflict, got %T: %v", err, err)
	}

	f, err := st.GetFlag(ctx, shop, "checkout-variant")
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}
	if f.ID == "" || f.Type != flag.Bool || len(f.Variations) != 2 {
		t.Fatalf("round-tripped flag looks wrong: %+v", f)
	}

	// configure prod: 10% canary
	if err := st.SetEnvConfig(ctx, flag.EnvConfig{
		FlagID:         f.ID,
		Env:            flag.EnvRef{Team: "alpha", Product: "shop", Stage: "prod"},
		Enabled:        true,
		DefaultVariant: "off",
		Rules:          []flag.Rule{{Rollout: []flag.RolloutBucket{{Variant: "on", Weight: 10}, {Variant: "off", Weight: 90}}}},
		UpdatedBy:      "josh",
	}); err != nil {
		t.Fatalf("SetEnvConfig: %v", err)
	}

	// EnvFlagSet + Project for prod -> ENABLED with fractional
	flags, cfgs, err := st.EnvFlagSet(ctx, flag.EnvRef{Team: "alpha", Product: "shop", Stage: "prod"})
	if err != nil {
		t.Fatalf("EnvFlagSet(prod): %v", err)
	}
	prod := flag.Project(flags, cfgs).Flags["checkout-variant"]
	if prod.State != "ENABLED" || prod.Targeting == nil {
		t.Fatalf("prod should be ENABLED with targeting, got %+v", prod)
	}

	// dev has no config -> DISABLED
	flags, cfgs, err = st.EnvFlagSet(ctx, flag.EnvRef{Team: "alpha", Product: "shop", Stage: "dev"})
	if err != nil {
		t.Fatalf("EnvFlagSet(dev): %v", err)
	}
	dev := flag.Project(flags, cfgs).Flags["checkout-variant"]
	if dev.State != "DISABLED" {
		t.Fatalf("dev should be DISABLED, got %q", dev.State)
	}

	// audit row written
	var n int
	if err := st.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action='set_config'`).Scan(&n); err != nil {
		t.Fatalf("audit count: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 audit row, got %d", n)
	}
}
