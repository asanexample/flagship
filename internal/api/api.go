// Package api is Flagship's control-plane management API: create/list/get/delete flag definitions and set
// per-Environment config. It is the write side that the dashboard (Phase 2) will drive; for now it's a
// clean REST surface over the Store. Auth (Keycloak RBAC, ADR-099 D2) is Phase 2 — handlers already take
// an actor for the audit trail.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/asanexample/flagship/internal/flag"
	"github.com/asanexample/flagship/internal/store"
)

// API holds the dependencies for the management handlers.
type API struct{ Store store.Store }

// Register wires the management routes onto a mux (Go 1.22+ method+path patterns).
func (a API) Register(mux *http.ServeMux) {
	const base = "/api/v1/teams/{team}/products/{product}/flags"
	mux.HandleFunc("POST "+base, a.createFlag)
	mux.HandleFunc("GET "+base, a.listFlags)
	mux.HandleFunc("GET "+base+"/{key}", a.getFlag)
	mux.HandleFunc("DELETE "+base+"/{key}", a.deleteFlag)
	mux.HandleFunc("PUT "+base+"/{key}/environments/{stage}", a.setConfig)
	mux.HandleFunc("GET "+base+"/{key}/environments/{stage}", a.getConfig)
}

func product(r *http.Request) flag.ProductRef {
	return flag.ProductRef{Team: r.PathValue("team"), Product: r.PathValue("product")}
}

type createFlagReq struct {
	Key         string         `json:"key"`
	Description string         `json:"description"`
	Type        flag.Type      `json:"type"`
	Variations  map[string]any `json:"variations"`
}

func (a API) createFlag(w http.ResponseWriter, r *http.Request) {
	var req createFlagReq
	if !decode(w, r, &req) {
		return
	}
	if req.Key == "" || len(req.Variations) == 0 {
		writeErr(w, http.StatusBadRequest, "key and at least one variation are required")
		return
	}
	f := flag.Flag{
		Product:     product(r),
		Key:         req.Key,
		Description: req.Description,
		Type:        req.Type,
		Variations:  req.Variations,
	}
	if err := a.Store.CreateFlag(r.Context(), f); err != nil {
		writeStoreErr(w, err)
		return
	}
	got, _ := a.Store.GetFlag(r.Context(), f.Product, f.Key)
	writeJSON(w, http.StatusCreated, got)
}

func (a API) listFlags(w http.ResponseWriter, r *http.Request) {
	flags, err := a.Store.ListFlags(r.Context(), product(r))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	if flags == nil {
		flags = []flag.Flag{}
	}
	writeJSON(w, http.StatusOK, flags)
}

func (a API) getFlag(w http.ResponseWriter, r *http.Request) {
	f, err := a.Store.GetFlag(r.Context(), product(r), r.PathValue("key"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (a API) deleteFlag(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.DeleteFlag(r.Context(), product(r), r.PathValue("key")); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type setConfigReq struct {
	Enabled        bool        `json:"enabled"`
	DefaultVariant string      `json:"defaultVariant"`
	Rules          []flag.Rule `json:"rules"`
	UpdatedBy      string      `json:"updatedBy"`
}

func (a API) setConfig(w http.ResponseWriter, r *http.Request) {
	var req setConfigReq
	if !decode(w, r, &req) {
		return
	}
	f, err := a.Store.GetFlag(r.Context(), product(r), r.PathValue("key"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	cfg := flag.EnvConfig{
		FlagID:         f.ID,
		Env:            flag.EnvRef{Team: f.Product.Team, Product: f.Product.Product, Stage: r.PathValue("stage")},
		Enabled:        req.Enabled,
		DefaultVariant: req.DefaultVariant,
		Rules:          req.Rules,
		UpdatedBy:      req.UpdatedBy,
	}
	if err := a.Store.SetEnvConfig(r.Context(), cfg); err != nil {
		writeStoreErr(w, err)
		return
	}
	// Return the persisted config (with the server-set updatedAt), not the request echo.
	saved, err := a.Store.GetEnvConfig(r.Context(), f.ID, cfg.Env)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (a API) getConfig(w http.ResponseWriter, r *http.Request) {
	f, err := a.Store.GetFlag(r.Context(), product(r), r.PathValue("key"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	cfg, err := a.Store.GetEnvConfig(r.Context(), f.ID, flag.EnvRef{Stage: r.PathValue("stage")})
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// --- helpers ---

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeStoreErr(w http.ResponseWriter, err error) {
	var nf store.ErrNotFound
	var cf store.ErrConflict
	switch {
	case errors.As(err, &nf):
		writeErr(w, http.StatusNotFound, err.Error())
	case errors.As(err, &cf):
		writeErr(w, http.StatusConflict, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, err.Error())
	}
}
