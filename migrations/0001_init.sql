-- Flagship control-plane schema (ADR-099 D2). Postgres 13+ (CNPG); gen_random_uuid is built-in.
--
-- Three tables:
--   flags       - Product-scoped flag DEFINITIONS (identity, type, variations)
--   env_configs - a flag's behaviour in ONE Environment (kill switch, default, ordered rules)
--   audit_log   - append-only record of every change (who/when/before/after)

BEGIN;

-- Product-scoped flag definitions. `variations` is {variantName: value}; values must match `type`
-- (enforced in the API layer, not the DB, since JSONB can't express "match this enum").
CREATE TABLE flags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team        TEXT        NOT NULL,
    product     TEXT        NOT NULL,
    key         TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    type        TEXT        NOT NULL CHECK (type IN ('boolean', 'string', 'number', 'json')),
    variations  JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team, product, key)
);

-- A flag's behaviour in one Environment (Product x Stage, ADR-067). `stage` alone identifies the
-- Environment because `flag_id` already implies (team, product). `rules` is the ordered []Rule as JSONB.
-- `enabled` is the top-level kill switch.
CREATE TABLE env_configs (
    flag_id         UUID        NOT NULL REFERENCES flags (id) ON DELETE CASCADE,
    stage           TEXT        NOT NULL,
    enabled         BOOLEAN     NOT NULL DEFAULT false,
    default_variant TEXT        NOT NULL,
    rules           JSONB       NOT NULL DEFAULT '[]',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by      TEXT        NOT NULL DEFAULT '',
    PRIMARY KEY (flag_id, stage)
);

-- The sync source reads all of an Environment's configs by stage; index for it.
CREATE INDEX env_configs_stage_idx ON env_configs (stage);

-- Append-only audit of every change. `before`/`after` are the JSON of the changed object (a flag def or
-- an env config); `stage` is null for flag-level actions.
CREATE TABLE audit_log (
    id       BIGSERIAL   PRIMARY KEY,
    at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor    TEXT        NOT NULL,
    team     TEXT        NOT NULL,
    product  TEXT        NOT NULL,
    flag_key TEXT        NOT NULL,
    stage    TEXT,
    action   TEXT        NOT NULL CHECK (action IN ('create_flag', 'delete_flag', 'set_config')),
    before   JSONB,
    after    JSONB
);

CREATE INDEX audit_log_scope_idx ON audit_log (team, product, flag_key, at DESC);

COMMIT;
