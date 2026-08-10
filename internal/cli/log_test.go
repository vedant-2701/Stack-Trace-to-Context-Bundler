package cli

import (
	"log/slog"
	"testing"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		verbosity int
		wantLevel slog.Level
	}{
		{name: "zero: default Warn", verbosity: 0, wantLevel: slog.LevelWarn},
		{name: "negative: still Warn, not below", verbosity: -1, wantLevel: slog.LevelWarn},
		{name: "one: Info", verbosity: 1, wantLevel: slog.LevelInfo},
		{name: "two: Debug", verbosity: 2, wantLevel: slog.LevelDebug},
		{name: "three or more: still Debug, capped", verbosity: 3, wantLevel: slog.LevelDebug},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := LogLevel(tc.verbosity)
			if got != tc.wantLevel {
				t.Errorf("LogLevel(%d) = %v, want %v", tc.verbosity, got, tc.wantLevel)
			}
		})
	}
}
