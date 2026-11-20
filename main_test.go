package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestMain lets tests run the real command-line entry point in a subprocess.
func TestMain(m *testing.M) {
	if os.Getenv("SESSION_LEDGER_TEST_MAIN") == "1" {
		main()
		return
	}
	os.Exit(m.Run())
}

type cli struct {
	stdout, stderr string
	code           int
}

// runCLI runs the program with an isolated home and agent stores. On Unix, PATH holds only a
// stub ps, so no real agent binary or process list influences the result.
func runCLI(t *testing.T, home string, args ...string) cli {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "SESSION_LEDGER_TEST_MAIN=1", "HOME="+home, "USERPROFILE="+home,
		"PI_CODING_AGENT_DIR="+filepath.Join(home, "omp"), "CLAUDE_CONFIG_DIR="+filepath.Join(home, "claude"),
		"CODEX_HOME="+filepath.Join(home, "codex"), "XDG_DATA_HOME="+filepath.Join(home, "data"))
	if runtime.GOOS != "windows" {
		bin := t.TempDir()
		writeFile(t, filepath.Join(bin, "ps"), "#!/bin/sh\necho init\n")
		if err := os.Chmod(filepath.Join(bin, "ps"), 0o755); err != nil {
			t.Fatal(err)
		}
		cmd.Env = append(cmd.Env, "PATH="+bin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	res := cli{stdout: out.String(), stderr: errb.String()}
	if ee, ok := err.(*exec.ExitError); ok {
		res.code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestCLIVersion(t *testing.T) {
	r := runCLI(t, t.TempDir(), "--version")
	if r.code != 0 || !strings.HasPrefix(r.stdout, "session-ledger ") || !strings.Contains(r.stdout, runtime.GOOS+"/"+runtime.GOARCH) {
		t.Errorf("%+v", r)
	}
}

func TestCLIRejectsBadArguments(t *testing.T) {
	for _, args := range [][]string{{"extra"}, {"--clean", "--preview"}, {"--version", "--clean"}, {"--version", "extra"}, {"--nope"}} {
		r := runCLI(t, t.TempDir(), args...)
		if r.code == 0 || r.stdout != "" || r.stderr == "" {
			t.Errorf("%v: %+v; want a nonzero exit with the reason on stderr", args, r)
		}
	}
}

func claudeFixture(t *testing.T) (home, transcript string) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := testUUID(1)
	transcript = filepath.Join(home, "claude", "projects", "proj", id+".jsonl")
	writeFile(t, transcript, record(t, map[string]any{"sessionId": id, "cwd": home})+"\n")
	return home, transcript
}

func TestCLIPreviewListsSessionsAndWritesNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on a stub process list")
	}
	home, transcript := claudeFixture(t)
	for _, args := range [][]string{nil, {"--preview"}} {
		r := runCLI(t, home, args...)
		if r.code != 0 || r.stderr != "" {
			t.Fatalf("%v: %+v", args, r)
		}
		for _, want := range []string{"Read-only. No files were moved.", "AGENT", "claude", home, "1 sessions · 1 eligible · 0 protected or unverified", "session-ledger --clean"} {
			if !strings.Contains(r.stdout, want) {
				t.Errorf("%v: output lacks %q:\n%s", args, want, r.stdout)
			}
		}
		if !exists(transcript) {
			t.Fatal("preview removed a transcript")
		}
	}
}

func TestCLIPreviewMarksUnverifiedSessionsAsKept(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on a stub process list")
	}
	home, transcript := claudeFixture(t)
	writeFile(t, transcript, "no header\n")
	r := runCLI(t, home, "--preview")
	if r.code != 0 || !strings.Contains(r.stdout, "(unverified project)") || !strings.Contains(r.stdout, "0 eligible · 1 protected or unverified") {
		t.Errorf("%+v", r)
	}
}

func TestCLICleanNeedsATerminalAndMovesNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on a stub process list")
	}
	home, transcript := claudeFixture(t)
	r := runCLI(t, home, "--clean")
	if r.code == 0 || !strings.Contains(r.stderr, "interactive cleanup requires a terminal") || !exists(transcript) {
		t.Errorf("%+v", r)
	}
	empty := t.TempDir()
	if r := runCLI(t, empty, "--clean"); r.code != 0 || !strings.Contains(r.stdout, "No local sessions found.") {
		t.Errorf("empty store: %+v", r)
	}
}

func TestPrintableEscapesControlCharacters(t *testing.T) {
	if got := printable("a\nb\x1b[31m"); strings.ContainsAny(got, "\n\x1b") {
		t.Errorf("control characters leaked: %q", got)
	}
	if label("") != "(unverified project)" || label("/work/a") != "/work/a" {
		t.Error("unexpected project labels")
	}
}

func press(m menu, keys ...tea.KeyMsg) (menu, tea.Cmd) {
	var cmd tea.Cmd
	for _, k := range keys {
		var next tea.Model
		next, cmd = m.Update(k)
		m = next.(menu)
	}
	return m, cmd
}

var (
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	down  = tea.KeyMsg{Type: tea.KeyDown}
	up    = tea.KeyMsg{Type: tea.KeyUp}
	left  = tea.KeyMsg{Type: tea.KeyLeft}
	space = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
)

func testMenu() menu {
	rows := []session{
		{agent: "claude", id: "a1", project: "/a"}, {agent: "claude", id: "a2", project: "/a"},
		{agent: "claude", id: "b1", project: "/b"}, {agent: "claude", id: "u1", project: ""},
		{agent: "omp", id: "o1", project: "/a"},
	}
	return menu{inv: inventory{rows: rows}, agents: []string{"claude", "omp"}, width: 80, height: 24}
}

func ids(rows []session) string {
	var s []string
	for _, r := range rows {
		s = append(s, r.id)
	}
	return strings.Join(s, ",")
}

func TestMenuSelectsMarkedProjectsOfTheChosenAgentOnly(t *testing.T) {
	m, _ := press(testMenu(), enter) // claude -> projects: "", /a, /b
	if m.stage != 1 || strings.Join(m.paths, "|") != "|/a|/b" {
		t.Fatalf("stage=%d paths=%q", m.stage, m.paths)
	}
	m, _ = press(m, down, space, down, space, enter)
	if m.stage != 2 || ids(m.selected) != "a1,a2,b1" {
		t.Fatalf("stage=%d selected=%s; want a1,a2,b1 (no omp row, no unmarked project)", m.stage, ids(m.selected))
	}
}

func TestMenuEnterWithoutMarksSelectsHighlightedProject(t *testing.T) {
	m, _ := press(testMenu(), enter, down, down, enter)
	if ids(m.selected) != "b1" {
		t.Errorf("selected = %s; want b1", ids(m.selected))
	}
}

func TestMenuAllProjectsAndIndividualProjectsAreMutuallyExclusive(t *testing.T) {
	m, _ := press(testMenu(), enter, down, space) // mark /a
	m, _ = press(m, down, down, space)            // move to "All projects" and mark it
	if len(m.marked) != 1 || !m.marked[3] {
		t.Fatalf("marking All projects should clear individual marks: %v", m.marked)
	}
	m, _ = press(m, up, space) // mark /b: clears All
	if len(m.marked) != 1 || !m.marked[2] {
		t.Fatalf("marking a project should clear All projects: %v", m.marked)
	}
	m, _ = press(m, enter)
	if ids(m.selected) != "b1" {
		t.Errorf("selected = %s", ids(m.selected))
	}
	all, _ := press(testMenu(), enter, down, down, down, space, enter)
	if ids(all.selected) != "a1,a2,b1,u1" {
		t.Errorf("All projects selected %s; want every claude row, including unverified", ids(all.selected))
	}
}

func TestMenuConfirmationDefaultsToNo(t *testing.T) {
	m, _ := press(testMenu(), enter, enter) // claude, first project
	m, cmd := press(m, enter)               // accept the default answer
	if m.confirmed || cmd == nil {
		t.Fatalf("default answer must cancel: confirmed=%v", m.confirmed)
	}
	if _, quit := cmd().(tea.QuitMsg); !quit {
		t.Error("confirmation should end the program")
	}
	m, _ = press(testMenu(), enter, enter, enter, down, enter)
	if !m.confirmed {
		t.Error("choosing Yes should confirm")
	}
}

func TestMenuNavigationWrapsGoesBackAndCancels(t *testing.T) {
	m, _ := press(testMenu(), up)
	if m.pos != 1 {
		t.Errorf("up from the first row should wrap to the last, pos=%d", m.pos)
	}
	m, _ = press(m, left)
	if m.stage != 0 {
		t.Error("left at the first stage must not move")
	}
	m, _ = press(m, enter, down, left)
	if m.stage != 0 || m.pos != 0 {
		t.Errorf("left should return to the agent list: stage=%d pos=%d", m.stage, m.pos)
	}
	m, cmd := press(m, esc)
	if _, quit := cmd().(tea.QuitMsg); !quit || m.confirmed {
		t.Error("esc must quit without confirming")
	}
}

func TestMenuViewShowsStageHintsAndFitsNarrowTerminals(t *testing.T) {
	m := testMenu()
	if v := m.View(); !strings.Contains(v, "Cleanup (1/3)") || strings.Contains(v, "Space Toggle") || strings.Contains(v, "Back") {
		t.Errorf("stage 0:\n%s", v)
	}
	m, _ = press(m, enter)
	m.width = 12
	v := m.View()
	if !strings.Contains(v, "Space Toggle") || !strings.Contains(v, "Back") || !strings.Contains(v, "[ ] ") {
		t.Errorf("stage 1:\n%s", v)
	}
	for _, line := range strings.Split(v, "\n")[2:5] {
		if len([]rune(line)) > 12 {
			t.Errorf("row not truncated to the terminal width: %q", line)
		}
	}
	m, _ = press(m, enter)
	m.width = 80
	if v := m.View(); !strings.Contains(v, "move eligible transcripts to Trash?") || !strings.Contains(v, "No, cancel") {
		t.Errorf("stage 2:\n%s", v)
	}
}
