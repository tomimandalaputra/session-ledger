package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestReadProject(t *testing.T) {
	proj := filepath.Join(t.TempDir(), "proj")
	sep := string(filepath.Separator)
	id := testUUID(1)
	lines := func(ls ...string) string { return strings.Join(ls, "\n") }
	omp := func(id, cwd string) string { return record(t, map[string]any{"id": id, "cwd": cwd}) }
	claude := func(id, cwd string) string { return record(t, map[string]any{"sessionId": id, "cwd": cwd}) }
	codex := func(typ, id, cwd string) string {
		return record(t, map[string]any{"type": typ, "payload": map[string]any{"id": id, "cwd": cwd}})
	}
	pad := func(n int) string { return record(t, map[string]any{"pad": strings.Repeat("a", n)}) }

	for _, tc := range []struct{ name, agent, data, want string }{
		{"omp match", "omp", omp(id, proj), proj},
		{"claude reads sessionId", "claude", claude(id, proj), proj},
		{"claude ignores plain id", "claude", omp(id, proj), ""},
		{"codex reads session_meta payload", "codex", codex("session_meta", id, proj), proj},
		{"codex ignores other record types", "codex", codex("event_msg", id, proj), ""},
		{"codex accepts a top-level id header", "codex", omp(id, proj), proj},
		{"id mismatch", "omp", omp(testUUID(2), proj), ""},
		{"relative cwd is unverified", "omp", omp(id, "relative/dir"), ""},
		{"cwd is cleaned", "omp", omp(id, proj+sep+"x"+sep+".."), proj},
		{"later matching record wins over a mismatch", "omp", lines(omp("other", proj+"x"), omp(id, proj)), proj},
		{"invalid JSON lines are skipped", "omp", lines("not json", "{", omp(id, proj)), proj},
		{"no trailing newline", "omp", omp(id, proj), proj},
		{"empty file", "omp", "", ""},
		{"large record under 1 MiB is parsed", "omp", record(t, map[string]any{"pad": strings.Repeat("a", 200<<10), "id": id, "cwd": proj}), proj},
		{"record over 1 MiB is skipped, not fatal", "omp", lines(pad(2<<20), omp(id, proj)), proj},
		{"only an oversized record", "omp", pad(2 << 20), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "s.jsonl")
			writeFile(t, path, tc.data)
			got, err := readProject(path, tc.agent, id)
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}

	if _, err := readProject(filepath.Join(t.TempDir(), "missing.jsonl"), "omp", id); err == nil {
		t.Error("missing file should return an error")
	}
}

func TestScanFindsOnlyTheDocumentedLayout(t *testing.T) {
	home := isolatedHome(t)
	ids := map[string]string{}
	for i, name := range []string{"omp", "ompArtifact", "ompDir", "ompNotes", "codex", "codexArchived", "codexShallow", "claude"} {
		ids[name] = testUUID(i + 1)
	}
	header := func(id string) string {
		return record(t, map[string]any{"id": id, "cwd": home})
	}
	codexHeader := func(id string) string {
		return record(t, map[string]any{"type": "session_meta", "payload": map[string]any{"id": id, "cwd": home}})
	}
	omp, codex, claude := filepath.Join(home, "omp", "sessions", "proj"), filepath.Join(home, "codex"), filepath.Join(home, "claude", "projects", "proj")

	writeFile(t, filepath.Join(omp, "2025-01-01_"+ids["omp"]+".jsonl"), header(ids["omp"]))
	writeFile(t, filepath.Join(omp, "artifacts", "2025-01-01_"+ids["ompArtifact"]+".jsonl"), header(ids["ompArtifact"]))
	if err := os.MkdirAll(filepath.Join(omp, "2025-01-01_"+ids["ompDir"]+".jsonl"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(omp, "notes.jsonl"), "{}")
	writeFile(t, filepath.Join(codex, "sessions", "2025", "01", "01", "rollout-x-"+ids["codex"]+".jsonl"), codexHeader(ids["codex"]))
	writeFile(t, filepath.Join(codex, "sessions", "2025", "rollout-x-"+ids["codexShallow"]+".jsonl"), codexHeader(ids["codexShallow"]))
	writeFile(t, filepath.Join(codex, "archived_sessions", "rollout-x-"+ids["codexArchived"]+".jsonl"), "no header here")
	writeFile(t, filepath.Join(claude, ids["claude"]+".jsonl"), record(t, map[string]any{"sessionId": ids["claude"], "cwd": home}))

	got := map[string]string{}
	for _, s := range scan(home).rows {
		got[s.id] = s.agent + "|" + s.project
	}
	want := map[string]string{
		ids["omp"]:           "omp|" + home,
		ids["codex"]:         "codex|" + home,
		ids["codexArchived"]: "codex|", // no verifiable header: listed, but never eligible
		ids["claude"]:        "claude|" + home,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s: got %q, want %q", id, got[id], w)
		}
	}
}

func TestScanSkipsLinksAndRelativeStores(t *testing.T) {
	home := isolatedHome(t)
	id := testUUID(1)
	real := filepath.Join(home, "elsewhere", "proj")
	writeFile(t, filepath.Join(real, id+".jsonl"), record(t, map[string]any{"sessionId": id, "cwd": home}))
	if err := os.MkdirAll(filepath.Join(home, "claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "elsewhere"), filepath.Join(home, "claude", "projects")); err != nil {
		t.Skipf("cannot create symlinks here: %v", err)
	}
	inv := scan(home)
	if len(inv.rows) != 0 || !slices.ContainsFunc(inv.warnings, func(w string) bool { return strings.HasPrefix(w, "Ignoring linked store") }) {
		t.Errorf("linked store should be ignored with a warning: rows=%d warnings=%v", len(inv.rows), inv.warnings)
	}

	// A symlinked transcript inside a real store is never followed.
	other := isolatedHome(t)
	linkDir := filepath.Join(other, "claude", "projects", "proj")
	if err := os.MkdirAll(linkDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(real, id+".jsonl"), filepath.Join(linkDir, id+".jsonl")); err != nil {
		t.Skipf("cannot create symlinks here: %v", err)
	}
	if rows := scan(other).rows; len(rows) != 0 {
		t.Errorf("symlinked transcript was inventoried: %+v", rows)
	}

	// A relative store location is refused rather than resolved against the working directory.
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join("rel", "projects"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "rel")
	inv = scan(other)
	if !slices.ContainsFunc(inv.warnings, func(w string) bool { return strings.HasPrefix(w, "Ignoring relative store") }) {
		t.Errorf("expected a relative-store warning, got %v", inv.warnings)
	}
}

func TestScanReportsDetectionOnlyAgentsWithoutCountingSessions(t *testing.T) {
	home := isolatedHome(t)
	t.Setenv("PATH", t.TempDir()) // no agent binaries visible
	for _, dir := range []string{filepath.Join(home, ".cursor", "chats"), filepath.Join(home, "data", "opencode")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	inv := scan(home)
	slices.Sort(inv.other)
	if want := []string{"Cursor Agent", "OpenCode"}; !slices.Equal(inv.other, want) {
		t.Errorf("other = %v, want %v", inv.other, want)
	}
	if len(inv.rows) != 0 || len(inv.agents) != 0 {
		t.Errorf("detection-only agents must not yield sessions: %+v", inv)
	}
}

func TestProjectsAreSortedDistinctAndPerAgent(t *testing.T) {
	rows := []session{{agent: "omp", project: "/b"}, {agent: "omp", project: "/a"}, {agent: "omp", project: "/b"}, {agent: "omp", project: ""}, {agent: "claude", project: "/z"}}
	if got, want := projects(rows, "omp"), []string{"", "/a", "/b"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := projects(rows, "codex"); len(got) != 0 {
		t.Errorf("unknown agent should have no projects, got %v", got)
	}
}

func TestUnchangedDetectsEveryKindOfDrift(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, s session)
		want   bool
	}{
		{"untouched", func(*testing.T, session) {}, true},
		{"content grew", func(t *testing.T, s session) { writeFile(t, s.path, "transcript with a new message appended\n") }, false},
		{"mtime moved", func(t *testing.T, s session) {
			later := s.info.ModTime().Add(time.Hour)
			if err := os.Chtimes(s.path, later, later); err != nil {
				t.Fatal(err)
			}
		}, false},
		{"removed", func(t *testing.T, s session) { os.Remove(s.path) }, false},
		{"replaced by another file of equal size and time", func(t *testing.T, s session) {
			tmp := s.path + ".new"
			writeFile(t, tmp, strings.Repeat("x", int(s.info.Size())))
			if err := os.Chtimes(tmp, s.info.ModTime(), s.info.ModTime()); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(tmp, s.path); err != nil {
				t.Fatal(err)
			}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			s := diskSession(t, "claude", filepath.Join(dir, "s.jsonl"), testUUID(1), dir)
			tc.mutate(t, s)
			if got := unchanged(s); got != tc.want {
				t.Errorf("unchanged = %v, want %v", got, tc.want)
			}
		})
	}
}
