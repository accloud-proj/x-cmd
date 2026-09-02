package updater

import "testing"

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v0.2.1", "0.2.0", true},
		{"v0.3.0", "0.2.9", true},
		{"v1.0.0", "0.99.99", true},
		{"v0.2.0", "0.2.0", false},
		{"v0.1.9", "0.2.0", false},
	}
	for _, test := range tests {
		if got := IsNewer(test.latest, test.current); got != test.want {
			t.Errorf("IsNewer(%q, %q) = %t", test.latest, test.current, got)
		}
	}
}
