package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/asanexample/flagship/internal/flag"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres is the production Store, backed by CNPG. Schema is in migrations/ and applied out-of-band (a
// migration Job / CI step), not on app start. Variations and rules are stored as JSONB.
type Postgres struct{ pool *pgxpool.Pool }

// NewPostgres opens a pooled connection and verifies it. dsn is a standard Postgres URL/keyword string.
func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

// Close releases the pool.
func (p *Postgres) Close() { p.pool.Close() }

func (p *Postgres) CreateFlag(ctx context.Context, f flag.Flag) error {
	vars, err := json.Marshal(f.Variations)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx,
		`INSERT INTO flags (team, product, key, description, type, variations)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		f.Product.Team, f.Product.Product, f.Key, f.Description, string(f.Type), vars)
	if isUniqueViolation(err) {
		return ErrConflict{What: "flag " + f.Key}
	}
	return err
}

func (p *Postgres) GetFlag(ctx context.Context, product flag.ProductRef, key string) (flag.Flag, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT id, description, type, variations, created_at
		 FROM flags WHERE team=$1 AND product=$2 AND key=$3`,
		product.Team, product.Product, key)
	f := flag.Flag{Product: product, Key: key}
	var typ string
	var vars []byte
	if err := row.Scan(&f.ID, &f.Description, &typ, &vars, &f.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return flag.Flag{}, ErrNotFound{What: "flag " + key}
		}
		return flag.Flag{}, err
	}
	f.Type = flag.Type(typ)
	if err := json.Unmarshal(vars, &f.Variations); err != nil {
		return flag.Flag{}, err
	}
	return f, nil
}

func (p *Postgres) ListFlags(ctx context.Context, product flag.ProductRef) ([]flag.Flag, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id, key, description, type, variations, created_at
		 FROM flags WHERE team=$1 AND product=$2 ORDER BY key`,
		product.Team, product.Product)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []flag.Flag
	for rows.Next() {
		f := flag.Flag{Product: product}
		var typ string
		var vars []byte
		if err := rows.Scan(&f.ID, &f.Key, &f.Description, &typ, &vars, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.Type = flag.Type(typ)
		if err := json.Unmarshal(vars, &f.Variations); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (p *Postgres) DeleteFlag(ctx context.Context, product flag.ProductRef, key string) error {
	// env_configs cascade via FK ON DELETE CASCADE.
	tag, err := p.pool.Exec(ctx, `DELETE FROM flags WHERE team=$1 AND product=$2 AND key=$3`,
		product.Team, product.Product, key)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound{What: "flag " + key}
	}
	return nil
}

func (p *Postgres) GetEnvConfig(ctx context.Context, flagID string, env flag.EnvRef) (flag.EnvConfig, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT enabled, default_variant, rules, updated_at, updated_by
		 FROM env_configs WHERE flag_id=$1 AND stage=$2`, flagID, env.Stage)
	cfg := flag.EnvConfig{FlagID: flagID, Env: env}
	var rules []byte
	if err := row.Scan(&cfg.Enabled, &cfg.DefaultVariant, &rules, &cfg.UpdatedAt, &cfg.UpdatedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return flag.EnvConfig{}, ErrNotFound{What: "config for " + flagID + "/" + env.Stage}
		}
		return flag.EnvConfig{}, err
	}
	if err := json.Unmarshal(rules, &cfg.Rules); err != nil {
		return flag.EnvConfig{}, err
	}
	return cfg, nil
}

// SetEnvConfig upserts the config and appends an audit record in a single transaction, so a change and
// its audit trail can never diverge (ADR-099 D2).
func (p *Postgres) SetEnvConfig(ctx context.Context, cfg flag.EnvConfig) error {
	rules, err := json.Marshal(cfg.Rules)
	if err != nil {
		return err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit

	// Capture the prior state for the audit "before".
	var before []byte
	_ = tx.QueryRow(ctx,
		`SELECT to_jsonb(c) FROM env_configs c WHERE flag_id=$1 AND stage=$2`,
		cfg.FlagID, cfg.Env.Stage).Scan(&before)

	tag, err := tx.Exec(ctx,
		`INSERT INTO env_configs (flag_id, stage, enabled, default_variant, rules, updated_by)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (flag_id, stage)
		 DO UPDATE SET enabled=$3, default_variant=$4, rules=$5, updated_at=now(), updated_by=$6`,
		cfg.FlagID, cfg.Env.Stage, cfg.Enabled, cfg.DefaultVariant, rules, cfg.UpdatedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound{What: "flag " + cfg.FlagID}
	}

	after, _ := json.Marshal(map[string]any{
		"enabled": cfg.Enabled, "defaultVariant": cfg.DefaultVariant, "rules": json.RawMessage(rules),
	})
	if _, err := tx.Exec(ctx,
		`INSERT INTO audit_log (actor, team, product, flag_key, stage, action, before, after)
		 SELECT $1, f.team, f.product, f.key, $2, 'set_config', $3, $4 FROM flags f WHERE f.id=$5`,
		cfg.UpdatedBy, cfg.Env.Stage, before, after, cfg.FlagID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// EnvFlagSet returns every flag in the Environment's Product plus that flag's config for the stage, in a
// single LEFT JOIN — the hot read behind the sync source.
func (p *Postgres) EnvFlagSet(ctx context.Context, env flag.EnvRef) ([]flag.Flag, map[string]flag.EnvConfig, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT f.id, f.key, f.description, f.type, f.variations,
		        c.enabled, c.default_variant, c.rules, c.updated_at, c.updated_by
		 FROM flags f
		 LEFT JOIN env_configs c ON c.flag_id = f.id AND c.stage = $3
		 WHERE f.team = $1 AND f.product = $2
		 ORDER BY f.key`,
		env.Team, env.Product, env.Stage)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var flags []flag.Flag
	cfgs := map[string]flag.EnvConfig{}
	for rows.Next() {
		f := flag.Flag{Product: flag.ProductRef{Team: env.Team, Product: env.Product}}
		var typ string
		var vars []byte
		// config columns are nullable (LEFT JOIN miss)
		var enabled *bool
		var defVar *string
		var rules []byte
		var updatedBy *string
		var updatedAt *any // read but unused in the sync path
		if err := rows.Scan(&f.ID, &f.Key, &f.Description, &typ, &vars,
			&enabled, &defVar, &rules, &updatedAt, &updatedBy); err != nil {
			return nil, nil, err
		}
		f.Type = flag.Type(typ)
		if err := json.Unmarshal(vars, &f.Variations); err != nil {
			return nil, nil, err
		}
		flags = append(flags, f)

		if enabled != nil { // a config row exists for this stage
			cfg := flag.EnvConfig{FlagID: f.ID, Env: env, Enabled: *enabled}
			if defVar != nil {
				cfg.DefaultVariant = *defVar
			}
			if updatedBy != nil {
				cfg.UpdatedBy = *updatedBy
			}
			if len(rules) > 0 {
				if err := json.Unmarshal(rules, &cfg.Rules); err != nil {
					return nil, nil, err
				}
			}
			cfgs[f.ID] = cfg
		}
	}
	return flags, cfgs, rows.Err()
}

// isUniqueViolation reports whether err is a Postgres unique-constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
