package cli

import "log/slog"

// LogLevel maps a verbosity count -- however many of -v/-vv main.go
// determined were set -- to the corresponding slog.Level. Shared by
// cmd/all, cmd/java, and cmd/typescript's main.go so verbosity behaves
// identically across all three binaries.
//
// 0 (neither flag set) is CONVENTIONS.md's "quiet by default" -- Warn.
// 1 (-v) is Info. 2 or more (-vv, or however main.go computes the count)
// is Debug, and stays Debug beyond that -- spec.md defines no level
// below Debug, so there's nothing further to escalate to.
//
// Pure function, no side effects: it does not touch slog's default
// logger. Wiring the returned Level into an actual handler
// (slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, ...)))) is
// main.go's job, not this helper's.
func LogLevel(verbosity int) slog.Level {
	switch {
	case verbosity <= 0:
		return slog.LevelWarn
	case verbosity == 1:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

// verbosityFromFlags converts the two -v/-vv boolean flags, as parsed by
// ParseAll/ParseFixedLang's FlagSet, into a single verbosity count
// matching LogLevel's expected input: 0 (neither), 1 (-v only), or 2
// (-vv, or both together). Max, not sum -- -v and -vv together still
// resolves to 2 (Debug), since nothing beyond Debug is defined.
func verbosityFromFlags(v, vv bool) int {
	switch {
	case vv:
		return 2
	case v:
		return 1
	default:
		return 0
	}
}
