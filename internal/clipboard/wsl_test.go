package clipboard

import (
	"errors"
	"testing"
)

func TestIsWSL(t *testing.T) {
	tests := []struct {
		name             string
		wslDistroNameSet bool
		wslDistroName    string
		wslInteropSet    bool
		wslInterop       string
		procVersion      string
		procVersionErr   error
		want             bool
	}{
		{
			name:             "WSL_DISTRO_NAME set (non-empty)",
			wslDistroNameSet: true,
			wslDistroName:    "Ubuntu-24.04",
			want:             true,
		},
		{
			name:          "WSL_INTEROP set (non-empty)",
			wslInteropSet: true,
			wslInterop:    "/run/WSL/1_interop",
			want:          true,
		},
		{
			// A present-but-empty env var must still count as "set"
			// (spec.md FR3 says "set," not "set to a non-empty value").
			// proc version deliberately has no microsoft mention, so a
			// pass here can only be explained by the env-var check
			// itself, not a fallback coincidence.
			name:          "WSL_INTEROP set but empty -- still counts as present",
			wslInteropSet: true,
			wslInterop:    "",
			procVersion:   "Linux version 5.15.0-generic (buildd@lcy02-amd64)",
			want:          true,
		},
		{
			name:        "proc version contains microsoft",
			procVersion: "Linux version 5.15.0 (Microsoft@Microsoft.com)",
			want:        true,
		},
		{
			name:        "proc version contains Microsoft, different case",
			procVersion: "Linux version 5.15.0 (SOME MICROSOFT BUILD)",
			want:        true,
		},
		{
			name:        "no env vars, proc version has no microsoft mention",
			procVersion: "Linux version 5.15.0-generic (buildd@lcy02-amd64)",
			want:        false,
		},
		{
			name:           "no env vars, proc version unreadable (non-Linux or missing)",
			procVersionErr: errors.New("no such file"),
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origLookupEnv := lookupEnv
			origReadProcVersion := readProcVersion
			t.Cleanup(func() {
				lookupEnv = origLookupEnv
				readProcVersion = origReadProcVersion
			})

			lookupEnv = func(key string) (string, bool) {
				switch key {
				case "WSL_DISTRO_NAME":
					if tt.wslDistroNameSet {
						return tt.wslDistroName, true
					}
					return "", false
				case "WSL_INTEROP":
					if tt.wslInteropSet {
						return tt.wslInterop, true
					}
					return "", false
				default:
					return "", false
				}
			}
			readProcVersion = func() ([]byte, error) {
				if tt.procVersionErr != nil {
					return nil, tt.procVersionErr
				}
				return []byte(tt.procVersion), nil
			}

			if got := isWSL(); got != tt.want {
				t.Errorf("isWSL() = %v, want %v", got, tt.want)
			}
		})
	}
}
