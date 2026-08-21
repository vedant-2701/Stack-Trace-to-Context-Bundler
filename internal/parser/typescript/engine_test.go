package typescript

import (
	"testing"

	"github.com/vedant-2701/stack-trace-bundler/internal/contract"
)

func TestParseFrameLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantOK    bool
		wantFrame contract.Frame
	}{
		{
			name:   "plain frame with function name",
			line:   "at Foo (bar.js:1:2)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "bar.js", LineNumber: 1, ColumnNumber: 2,
				MethodName: "Foo",
			},
		},
		{
			name:   "bare frame, no function name",
			line:   "at bar.js:1:2",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "bar.js", LineNumber: 1, ColumnNumber: 2,
			},
		},
		{
			name:   "non-matching line is not an error",
			line:   "this is not an error header just random text",
			wantOK: false,
		},
		{
			name:   "system-error property line is not a frame (FR7)",
			line:   "  errno: -111,",
			wantOK: false,
		},
		{
			name:   "top-level property line is not a frame (FR7)",
			line:   "  code: 'GenericFailure'",
			wantOK: false,
		},
		{
			name:   "6-space-indented nested-cause frame",
			line:   "      at level2 (/home/vedant/script.js:5:10)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "/home/vedant/script.js", LineNumber: 5, ColumnNumber: 10,
				MethodName: "level2",
			},
		},
		{
			name:   "bare frame with trailing brace before [cause]:",
			line:   "    at node:internal/main/run_main_module:33:47 {",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "node:internal/main/run_main_module", LineNumber: 33, ColumnNumber: 47,
			},
		},
		{
			name:   "described frame with trailing brace before [cause]:",
			line:   "      at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1706:16) {",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "node:net", LineNumber: 1706, ColumnNumber: 16,
				ClassName: "TCPConnectWrap", MethodName: "afterConnect [as oncomplete]",
			},
		},
		{
			name:   "async bare frame",
			line:   "    at async node:internal/modules/esm/loader:643:26",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "node:internal/modules/esm/loader", LineNumber: 643, ColumnNumber: 26,
			},
		},
		{
			name:   "async described frame",
			line:   "    at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:101:5)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "node:internal/modules/run_main", LineNumber: 101, ColumnNumber: 5,
				MethodName: "asyncRunEntryPointWithESMLoader",
			},
		},
		{
			name:   "dot-qualified description splits into ClassName/MethodName",
			line:   "    at Object.<anonymous> (/home/vedant/script.js:7:15)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath: "/home/vedant/script.js", LineNumber: 7, ColumnNumber: 15,
				ClassName: "Object", MethodName: "<anonymous>",
			},
		},
		{
			name:   "file:// URI frame path is preserved, not yet normalized",
			line:   "    at main (file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js:4:9)",
			wantOK: true,
			wantFrame: contract.Frame{
				FilePath:   "file:///home/vedant/stack-trace-bundler/errors-test/pg-fails.js",
				LineNumber: 4, ColumnNumber: 9,
				MethodName: "main",
			},
		},
		{
			name:   "blank line is not a frame",
			line:   "",
			wantOK: false,
		},
		{
			name:   "closing brace alone is not a frame",
			line:   "}",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrame, gotOK := parseFrameLine(tt.line)
			if gotOK != tt.wantOK {
				t.Fatalf("parseFrameLine(%q) ok = %v, want %v", tt.line, gotOK, tt.wantOK)
			}
			if !gotOK {
				return
			}
			if gotFrame != tt.wantFrame {
				t.Errorf("parseFrameLine(%q) = %+v, want %+v", tt.line, gotFrame, tt.wantFrame)
			}
		})
	}
}
