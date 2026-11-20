package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBusyMatchesAgentNamesInCommandsNotSubstrings(t *testing.T) {
	for _, tc := range []struct {
		name, agent, ps string
		want            bool
	}{
		{"executable", "claude", "claude /home/u/.local/bin/claude --resume", true},
		{"node launcher script", "codex", "node /usr/lib/node_modules/@openai/codex/bin/codex.js", true},
		{"claude-code package", "claude", "node /opt/claude-code/cli", true},
		{"windows path", "claude", `claude.exe C:\Program Files\claude\claude.exe`, true},
		{"quoted argument", "omp", `sh -c "omp session"`, true},
		{"longer word is not the agent", "claude", "bash /home/u/claudette.sh", false},
		{"substring of a file name", "codex", "vim codex-notes.txt", false},
		{"unrelated process", "omp", "init /sbin/init", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fakeBin(t, "ps", fmt.Sprintf("cat <<'EOF'\n%s\nEOF", tc.ps))
			got, err := busy(tc.agent)
			if err != nil || got != tc.want {
				t.Errorf("busy = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}

func TestBusyTreatsUnverifiableProcessListAsBusy(t *testing.T) {
	for name, script := range map[string]string{"ps fails": "exit 1", "ps prints nothing": "exit 0"} {
		t.Run(name, func(t *testing.T) {
			fakeBin(t, "ps", script)
			got, err := busy("claude")
			if !got || err == nil {
				t.Errorf("busy = %v, %v; want true with an error", got, err)
			}
		})
	}
}

func TestEligibleForClaudeAndCodexDependsOnProcessList(t *testing.T) {
	rows := []session{{agent: "claude", id: "a", project: "/p"}, {agent: "claude", id: "b", project: ""}, {agent: "omp", id: "c", project: "/p"}}

	fakeBin(t, "ps", "echo init")
	ok, err := eligible("claude", rows)
	if err != nil || len(ok) != 1 || !ok["a"] {
		t.Errorf("idle: ok=%v err=%v; want only a (unverified and other-agent rows excluded)", ok, err)
	}

	fakeBin(t, "ps", "echo claude --resume")
	if ok, err := eligible("claude", rows); err == nil || !strings.Contains(err.Error(), "close claude") || ok != nil {
		t.Errorf("running: ok=%v err=%v; want a close-the-agent error", ok, err)
	}

	fakeBin(t, "ps", "exit 1")
	if _, err := eligible("claude", rows); err == nil {
		t.Error("unverifiable process list must block cleanup")
	}
	if ok, err := eligible("claude", []session{{agent: "claude", id: "x", project: ""}}); err != nil || len(ok) != 0 {
		t.Errorf("nothing to verify should not consult ps: ok=%v err=%v", ok, err)
	}
}

// fakeOmp installs an omp that marks ids starting with "held-" as held and "unknown-" as
// unknown, and appends one line per invocation to the returned log.
func fakeOmp(t *testing.T) string {
	t.Helper()
	log := filepath.Join(t.TempDir(), "calls")
	fakeBin(t, "omp", fmt.Sprintf(`echo "$#" >> %s
[ "$1 $2 $3 $4" = "session lease --json --" ] || exit 2
shift 4
held=""; unk=""
for id; do case $id in held-*) held="$held\"$id\",";; unknown-*) unk="$unk\"$id\",";; esac; done
printf '{"held":[%%s],"unknown":[%%s]}\n' "${held%%,}" "${unk%%,}"`, log))
	return log
}

func calls(t *testing.T, log string) []string {
	t.Helper()
	b, err := os.ReadFile(log)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(b))
}

func TestEligibleForOmpUsesLeaseAndExcludesHeldAndUnknown(t *testing.T) {
	log := fakeOmp(t)
	rows := []session{
		{agent: "omp", id: "free-1", project: "/p"},
		{agent: "omp", id: "held-1", project: "/p"},
		{agent: "omp", id: "unknown-1", project: "/p"},
		{agent: "omp", id: "free-unverified", project: ""},
		{agent: "claude", id: "free-other-agent", project: "/p"},
	}
	ok, err := eligible("omp", rows)
	if err != nil || len(ok) != 1 || !ok["free-1"] {
		t.Fatalf("ok=%v err=%v; want only free-1", ok, err)
	}
	// Ids reach omp as arguments after "--", and only verified omp rows are sent.
	if got := calls(t, log); len(got) != 1 || got[0] != "7" {
		t.Errorf("omp calls (arg counts) = %v; want one call with 4 flags + 3 ids", got)
	}
}

func TestEligibleForOmpBatchesLargeInventories(t *testing.T) {
	log := fakeOmp(t)
	var rows []session
	for i := range 250 {
		rows = append(rows, session{agent: "omp", id: fmt.Sprintf("free-%d", i), project: "/p"})
	}
	rows = append(rows, session{agent: "omp", id: "held-last", project: "/p"})
	ok, err := eligible("omp", rows)
	if err != nil || len(ok) != 250 || ok["held-last"] {
		t.Fatalf("len(ok)=%d err=%v; want the 250 free sessions", len(ok), err)
	}
	if got := calls(t, log); len(got) != 3 || got[0] != "104" || got[2] != "55" {
		t.Errorf("arg counts per call = %v; want batches of 100, 100, 51 ids", got)
	}
}

func TestEligibleForOmpFailsClosed(t *testing.T) {
	rows := []session{{agent: "omp", id: "free-1", project: "/p"}}
	for name, script := range map[string]string{
		"lease command fails":  "exit 3",
		"not JSON":             "echo garbage",
		"missing unknown list": `echo '{"held":[]}'`,
		"missing held list":    `echo '{"unknown":[]}'`,
		"null lists":           `echo '{"held":null,"unknown":null}'`,
		"empty output":         "exit 0",
	} {
		t.Run(name, func(t *testing.T) {
			fakeBin(t, "omp", script)
			if ok, err := eligible("omp", rows); err == nil || ok != nil {
				t.Errorf("ok=%v err=%v; want an error and no eligible sessions", ok, err)
			}
		})
	}
}

func TestUnchangedRejectsSymlinkedPath(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := diskSession(t, "claude", filepath.Join(dir, "s.jsonl"), testUUID(1), dir)
	target := filepath.Join(dir, "target.jsonl")
	writeFile(t, target, "transcript "+testUUID(1)+"\n")
	if err := os.Remove(s.path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, s.path); err != nil {
		t.Skipf("cannot create symlinks here: %v", err)
	}
	if unchanged(s) {
		t.Error("a path swapped for a symlink must not be treated as unchanged")
	}
}

// fakeGio installs a gio that moves files into trash, failing for paths containing "fail".
func fakeGio(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("exercises the Linux gio trash path")
	}
	trashDir := filepath.Join(t.TempDir(), "trash")
	if err := os.MkdirAll(trashDir, 0o700); err != nil {
		t.Fatal(err)
	}
	fakeBin(t, "gio", fmt.Sprintf(`[ "$1 $2" = "trash --" ] || exit 2
case "$3" in *fail*) echo "cannot trash" >&2; exit 1;; esac
mv "$3" %s/`, trashDir))
	return trashDir
}

func TestCleanupMovesOnlyVerifiedUnchangedTranscripts(t *testing.T) {
	trashDir := fakeGio(t)
	fakeBin(t, "ps", "echo init")
	store, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(store, "proj")
	moved1 := diskSession(t, "claude", filepath.Join(proj, "moved-1.jsonl"), "moved-1", store)
	moved2 := diskSession(t, "claude", filepath.Join(proj, "moved-2.jsonl"), "moved-2", store)
	changed := diskSession(t, "claude", filepath.Join(proj, "changed.jsonl"), "changed", store)
	unverified := diskSession(t, "claude", filepath.Join(proj, "unverified.jsonl"), "unverified", "")
	writeFile(t, changed.path, "a message arrived after selection\n")
	sibling := filepath.Join(proj, "moved-1", "subagents", "agent-1.jsonl")
	writeFile(t, sibling, "artifact\n")
	writeFile(t, filepath.Join(store, "settings.json"), "{}")

	moved, kept, err := cleanup([]session{moved1, changed, unverified, moved2})
	if err != nil || moved != 2 || kept != 2 {
		t.Fatalf("moved=%d kept=%d err=%v; want 2, 2, nil", moved, kept, err)
	}
	for _, s := range []session{moved1, moved2} {
		if exists(s.path) || !exists(filepath.Join(trashDir, filepath.Base(s.path))) {
			t.Errorf("%s was not moved to trash", s.id)
		}
	}
	for _, p := range []string{changed.path, unverified.path, sibling, filepath.Join(store, "settings.json")} {
		if !exists(p) {
			t.Errorf("%s must stay in place", p)
		}
	}
}

func TestCleanupStopsAtFirstTrashFailureAndReportsProgress(t *testing.T) {
	trashDir := fakeGio(t)
	fakeBin(t, "ps", "echo init")
	store, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var rows []session
	for _, name := range []string{"ok-1", "fail-2", "ok-3"} {
		rows = append(rows, diskSession(t, "claude", filepath.Join(store, name+".jsonl"), name, store))
	}
	moved, kept, err := cleanup(rows)
	if err == nil || !strings.Contains(err.Error(), "fail-2.jsonl") || moved != 1 || kept != 2 {
		t.Fatalf("moved=%d kept=%d err=%v; want 1, 2, error naming fail-2.jsonl", moved, kept, err)
	}
	if !exists(filepath.Join(trashDir, "ok-1.jsonl")) || !exists(rows[1].path) || !exists(rows[2].path) {
		t.Error("batch must stop after the first failure and leave later transcripts alone")
	}
}

func TestCleanupSkipsEverythingWhenAgentIsRunning(t *testing.T) {
	fakeGio(t)
	fakeBin(t, "ps", "echo node /opt/codex/bin/codex.js")
	store, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rows := []session{diskSession(t, "codex", filepath.Join(store, "a.jsonl"), "a", store)}
	moved, kept, err := cleanup(rows)
	if err == nil || moved != 0 || kept != 1 || !exists(rows[0].path) {
		t.Errorf("moved=%d kept=%d err=%v; want nothing moved while codex runs", moved, kept, err)
	}
}

func TestCleanupSkipsOmpSessionsThatAreHeld(t *testing.T) {
	trashDir := fakeGio(t)
	fakeOmp(t)
	store, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	free := diskSession(t, "omp", filepath.Join(store, "free-1.jsonl"), "free-1", store)
	held := diskSession(t, "omp", filepath.Join(store, "held-1.jsonl"), "held-1", store)
	moved, kept, err := cleanup([]session{free, held})
	if err != nil || moved != 1 || kept != 1 || !exists(held.path) || !exists(filepath.Join(trashDir, "free-1.jsonl")) {
		t.Errorf("moved=%d kept=%d err=%v; want only the unheld session moved", moved, kept, err)
	}
}

func TestCleanupWithNothingSelectedDoesNothing(t *testing.T) {
	if moved, kept, err := cleanup(nil); moved != 0 || kept != 0 || err != nil {
		t.Errorf("got %d, %d, %v", moved, kept, err)
	}
}

func TestTrashReportsWhenFileIsStillThere(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("exercises the Linux gio trash path")
	}
	fakeBin(t, "gio", "exit 0") // claims success without moving anything
	path := filepath.Join(t.TempDir(), "s.jsonl")
	writeFile(t, path, "x")
	if err := trash(path); err == nil || !strings.Contains(err.Error(), "did not move") {
		t.Errorf("err = %v; want a did-not-move error", err)
	}
	fakeBin(t, "gio", "echo no trash can >&2; exit 1")
	if err := trash(path); err == nil || !strings.Contains(err.Error(), "no trash can") {
		t.Errorf("err = %v; want gio's message in the error", err)
	}
}
