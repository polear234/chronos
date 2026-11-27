package main

import "testing"

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"0.3.3", "0.3.2", true},
		{"0.3.2", "0.3.2", false},
		{"0.4.0", "0.3.9", true},
		{"1.0.0", "0.9.9", true},
		{"0.3.2", "0.4.0", false},
		{"v0.3.3", "0.3.2", true},
		{"0.3.3-rc.1", "0.3.2", true},
	}
	for _, tt := range tests {
		if got := isNewer(tt.latest, tt.current); got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestAssetNameFor(t *testing.T) {
	tests := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "chpp-linux-amd64"},
		{"linux", "arm64", "chpp-linux-arm64"},
		{"windows", "amd64", "chpp-windows-amd64.exe"},
		{"darwin", "arm64", "chpp-darwin-arm64"},
	}
	for _, tt := range tests {
		if got := assetNameFor(tt.goos, tt.goarch); got != tt.want {
			t.Errorf("assetNameFor(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
		}
	}
}
