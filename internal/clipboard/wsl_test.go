package clipboard

import (
	"errors"
	"testing"
)

func TestIsWSL(t *testing.T) {
	tests := []struct {
		name           string
		wslDistroName  string
		wslInterop     string
		procVersion    string
		procVersionErr error
		want           bool
	}{
		{
			name:          "WSL_DISTRO_NAME set",
			wslDistroName: "Ubuntu-24.04",
			want:          true,
		},
		{
			name:       "WSL_INTEROP set",
			wslInterop: "/run/WSL/1_interop",
			want:       true,
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
			origGetenv := getenv
			origReadProcVersion := readProcVersion
			t.Cleanup(func() {
				getenv = origGetenv
				readProcVersion = origReadProcVersion
			})

			getenv = func(key string) string {
				switch key {
				case "WSL_DISTRO_NAME":
					return tt.wslDistroName
				case "WSL_INTEROP":
					return tt.wslInterop
				default:
					return ""
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
