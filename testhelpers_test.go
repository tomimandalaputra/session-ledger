package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testUUID(n int) string { return fmt.Sprintf("c0ffee00-0000-4000-8000-%012d", n) }

// isolatedHome points every agent store at a fresh temporary directory so no test
// can see or touch real session history.
func isolatedHome(t *testing.T) string {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for key, dir := range map[string]string{"PI_CODING_AGENT_DIR": "omp", "CLAUDE_CONFIG_DIR": "claude", "CODEX_HOME": "codex", "XDG_DATA_HOME": "data"} {
		t.Setenv(key, filepath.Join(home, dir))
	}
	return home
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func record(t *testing.T, v map[string]any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// diskSession writes a transcript and returns the session value discovery would build.
func diskSession(t *testing.T, agent, path, id, project string) session {
	t.Helper()
	writeFile(t, path, "transcript "+id+"\n")
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return session{agent: agent, path: path, id: id, project: project, info: info}
}

// fakeBin puts a shell script named name first on PATH. Skipped on Windows.
func fakeBin(t *testing.T, name, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell script as a stand-in executable")
	}
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, name), "#!/bin/sh\n"+script+"\n")
	if err := os.Chmod(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
