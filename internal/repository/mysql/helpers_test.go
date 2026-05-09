//go:build integration

package mysql_test

import (
	"io"
	"log/slog"
)

// testLogger returns a discard logger for integration tests so that repository
// signatures requiring *slog.Logger can be satisfied without leaking output.
func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
