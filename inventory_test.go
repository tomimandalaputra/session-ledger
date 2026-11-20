package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestInventoryRequiresMatchingIDAndProtectsChangedFiles(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := "c0ffee00-0000-4000-8000-000000000001"
	for key, dir := range map[string]string{"PI_CODING_AGENT_DIR": "omp", "CLAUDE_CONFIG_DIR": "claude", "CODEX_HOME": "codex"} {
		t.Setenv(key, filepath.Join(home, dir))
	}
	for _, tc := range []struct{ agent, relative, header string }{
		{"omp", "omp/sessions/project/2025-01-01_" + id + ".jsonl", `{"id":%q,"cwd":%q}`},
		{"claude", "claude/projects/project/" + id + ".jsonl", `{"sessionId":%q,"cwd":%q}`},
		{"codex", "codex/sessions/2025/01/01/rollout-date-" + id + ".jsonl", `{"type":"session_meta","payload":{"id":%q,"cwd":%q}}`},
	} {
		path := filepath.Join(home, filepath.FromSlash(tc.relative))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		data := fmt.Sprintf(tc.header, "not-the-session", filepath.Join(home, "wrong")) + "\n" + fmt.Sprintf(tc.header, id, home)
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, s := range scan(home).rows {
			if s.path != path {
				continue
			}
			found = true
			if s.agent != tc.agent || s.project != home || !unchanged(s) {
				t.Fatalf("bad inventory: %+v", s)
			}
			if err := os.WriteFile(path, []byte("changed after selection"), 0600); err != nil {
				t.Fatal(err)
			}
			if unchanged(s) {
				t.Fatal("changed transcript still eligible")
			}
		}
		if !found {
			t.Errorf("missing %s transcript", tc.agent)
		}
	}
}
