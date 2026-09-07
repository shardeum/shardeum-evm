package integration

import (
	"os"
	"path/filepath"
	"runtime"
)

// init configures the environment that shardeum's strict GetBlockGasLimit
// (config/server_app_options.go) requires when a ShardeumApp is constructed:
// SHARDEUM_NETWORK selects the network, and SHARDEUM_CONFIG_DIR points at the
// directory holding the network genesis files.
//
// This lives in a test-support package that production never imports, so it
// only runs inside test binaries. Production keeps its fail-fast behavior:
// if SHARDEUM_NETWORK is unset on a real node, GetBlockGasLimit still panics.
//
// Externally-set values are respected, so a CI run can still override the
// network or config directory.
func init() {
	if os.Getenv("SHARDEUM_NETWORK") == "" {
		_ = os.Setenv("SHARDEUM_NETWORK", "local")
	}
	if os.Getenv("SHARDEUM_CONFIG_DIR") == "" {
		// this file lives at <repo>/shardeumd/tests/integration/env_setup.go
		_, thisFile, _, _ := runtime.Caller(0)
		repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
		_ = os.Setenv("SHARDEUM_CONFIG_DIR", filepath.Join(repoRoot, "config"))
	}
}
