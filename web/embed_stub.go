//go:build !dashboard

package web

import "io/fs"

// No dashboard embedded (built without -tags dashboard). The server runs API + sync only.
func dashboardFS() (fs.FS, bool) { return nil, false }
