package main

import "testing"

func TestBuildVersionPrefersLinkerValueAndDropsLeadingV(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })
	for in, want := range map[string]string{"0.0.1": "0.0.1", "v1.2.3-rc.1": "1.2.3-rc.1"} {
		version = in
		if got := buildVersion(); got != want {
			t.Errorf("version %q: got %q, want %q", in, got, want)
		}
	}
}
