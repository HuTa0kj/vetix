package plugin

import "testing"

func TestFingerprintIsStable(t *testing.T) {
	a, b := Fingerprint(), Fingerprint()
	if a == "" {
		t.Fatal("Fingerprint returned empty string")
	}
	if len(a) != 16 {
		t.Fatalf("Fingerprint length = %d, want 16", len(a))
	}
	if a != b {
		t.Fatalf("Fingerprint not stable: %q != %q", a, b)
	}
}

func TestFingerprintTracksPluginSet(t *testing.T) {
	before := Fingerprint()

	old := registry
	registry = []Plugin{probePlugin{}}
	defer func() { registry = old }()

	after := Fingerprint()
	if before == after {
		t.Fatal("Fingerprint unchanged after changing the plugin set")
	}
}

type probePlugin struct{}

func (probePlugin) Meta() Meta {
	return Meta{ID: "probe", Name: "Probe", Description: "test-only"}
}

func (probePlugin) Scan(string, string, string) []Issue { return nil }
