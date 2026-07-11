package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/asanexample/flagship/internal/flag"
)

// Memory is an in-process Store for local development and unit tests. The Postgres implementation
// (Phase 1 next) is the production one; this keeps the API and sync source runnable + testable with no
// database. Flag IDs are the deterministic "team/product/key" so tests are stable.
type Memory struct {
	mu      sync.RWMutex
	flags   map[string]flag.Flag      // id -> flag
	configs map[string]flag.EnvConfig // id + "\x00" + stage -> config
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{flags: map[string]flag.Flag{}, configs: map[string]flag.EnvConfig{}}
}

func flagID(p flag.ProductRef, key string) string { return p.Team + "/" + p.Product + "/" + key }
func cfgKey(flagID, stage string) string          { return flagID + "\x00" + stage }

func (m *Memory) CreateFlag(_ context.Context, f flag.Flag) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := flagID(f.Product, f.Key)
	if _, exists := m.flags[id]; exists {
		return ErrConflict{What: "flag " + f.Key}
	}
	f.ID = id
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now().UTC()
	}
	m.flags[id] = f
	return nil
}

func (m *Memory) GetFlag(_ context.Context, p flag.ProductRef, key string) (flag.Flag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.flags[flagID(p, key)]
	if !ok {
		return flag.Flag{}, ErrNotFound{What: "flag " + key}
	}
	return f, nil
}

func (m *Memory) ListFlags(_ context.Context, p flag.ProductRef) ([]flag.Flag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []flag.Flag
	for _, f := range m.flags {
		if f.Product == p {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (m *Memory) DeleteFlag(_ context.Context, p flag.ProductRef, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := flagID(p, key)
	if _, ok := m.flags[id]; !ok {
		return ErrNotFound{What: "flag " + key}
	}
	delete(m.flags, id)
	for k := range m.configs {
		if len(k) > len(id) && k[:len(id)] == id {
			delete(m.configs, k)
		}
	}
	return nil
}

func (m *Memory) GetEnvConfig(_ context.Context, flagID string, env flag.EnvRef) (flag.EnvConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.configs[cfgKey(flagID, env.Stage)]
	if !ok {
		return flag.EnvConfig{}, ErrNotFound{What: "config for " + flagID + "/" + env.Stage}
	}
	return c, nil
}

func (m *Memory) SetEnvConfig(_ context.Context, cfg flag.EnvConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.flags[cfg.FlagID]; !ok {
		return ErrNotFound{What: "flag " + cfg.FlagID}
	}
	cfg.UpdatedAt = time.Now().UTC()
	m.configs[cfgKey(cfg.FlagID, cfg.Env.Stage)] = cfg
	return nil
}

func (m *Memory) EnvFlagSet(_ context.Context, env flag.EnvRef) ([]flag.Flag, map[string]flag.EnvConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p := flag.ProductRef{Team: env.Team, Product: env.Product}
	var flags []flag.Flag
	cfgs := map[string]flag.EnvConfig{}
	for _, f := range m.flags {
		if f.Product != p {
			continue
		}
		flags = append(flags, f)
		if c, ok := m.configs[cfgKey(f.ID, env.Stage)]; ok {
			cfgs[f.ID] = c
		}
	}
	sort.Slice(flags, func(i, j int) bool { return flags[i].Key < flags[j].Key })
	return flags, cfgs, nil
}
