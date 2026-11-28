package main

import "testing"

func TestVersionSatisfies(t *testing.T) {
	tests := []struct {
		version, constraint string
		want                bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1.0.0", "1.0.1", false},
		{"1.2.0", ">=1.0.0", true},
		{"0.9.0", ">=1.0.0", false},
		{"1.0.0", "<2.0.0", true},
		{"2.5.0", "<2.0.0", false},
		{"1.5.0", "^1.0.0", true},   // ^1.x 允许
		{"2.0.0", "^1.0.0", false},  // ^1.x 不允许 2.x
		{"1.2.5", "~1.2.0", true},   // ~1.2.x 允许
		{"1.3.0", "~1.2.0", false},  // ~1.2.x 不允许 1.3
		{"1.0.0", "*", true},
		{"1.0.0", "", true},
	}
	for _, tt := range tests {
		if got := versionSatisfies(tt.version, tt.constraint); got != tt.want {
			t.Errorf("versionSatisfies(%q, %q) = %v, want %v",
				tt.version, tt.constraint, got, tt.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"v1.2.3", "1.2.3", 0},
		{"1.2.3-rc.1", "1.2.3", 0},
	}
	for _, tt := range tests {
		if got := compareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
