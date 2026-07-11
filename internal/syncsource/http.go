// Package syncsource serves an Environment's flags to evaluators in flagd's flag-set schema.
//
// v1 is a flagd "http" sync source: an evaluator (a flagd sidecar/standalone, or an in-process provider)
// fetches GET /sync/{team}/{product}/{stage} to load a Product's flags for one Stage, and re-fetches on
// flagd's poll interval. The ~ms push upgrade — flagd's gRPC FlagSyncService streamed through a
// per-cluster relay, Environment-scoped and mutual-auth-authenticated — is ADR-099 D5 / Phase 3.
package syncsource

import (
	"encoding/json"
	"net/http"

	"github.com/asanexample/flagship/internal/flag"
	"github.com/asanexample/flagship/internal/store"
)

// HTTP serves the projected flagd flag-set for an Environment.
type HTTP struct{ Store store.Store }

// Register wires the sync route onto a mux.
func (h HTTP) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /sync/{team}/{product}/{stage}", h.serve)
}

func (h HTTP) serve(w http.ResponseWriter, r *http.Request) {
	// The URL is already Environment-scoped, so a consumer only ever fetches its own Product/Stage set.
	// Enforcing that a caller may ONLY fetch its own Environment (mutual-auth identity → allowed scope)
	// is Phase 3; today the scoping is by convention + network policy.
	env := flag.EnvRef{
		Team:    r.PathValue("team"),
		Product: r.PathValue("product"),
		Stage:   r.PathValue("stage"),
	}
	flags, cfgs, err := h.Store.EnvFlagSet(r.Context(), env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// flag.Project turns our model into flagd's schema — the evaluator does the rest.
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(flag.Project(flags, cfgs))
}
