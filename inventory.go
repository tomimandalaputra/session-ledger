package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type session struct {
	agent, path, id, project string
	info                     os.FileInfo
}
type inventory struct {
	rows                    []session
	agents, other, warnings []string
}

const uuid = `[0-9a-fA-F]{8}(?:-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}`

var filenames = map[string]*regexp.Regexp{"omp": regexp.MustCompile(`^\d{4}-.+_(` + uuid + `)\.jsonl$`), "claude": regexp.MustCompile(`^(` + uuid + `)\.jsonl$`), "codex": regexp.MustCompile(`^rollout-.+-(` + uuid + `)\.jsonl$`)}

func setting(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}
func directory(path string) bool { st, e := os.Stat(path); return e == nil && st.IsDir() }
func installed(tool string) bool { _, e := exec.LookPath(tool); return e == nil }
func scan(home string) inventory {
	var inv inventory
	for _, a := range []struct {
		name, root string
		patterns   []string
	}{
		{"omp", filepath.Join(setting("PI_CODING_AGENT_DIR", filepath.Join(home, ".omp", "agent")), "sessions"), []string{"*/*.jsonl"}},
		{"claude", filepath.Join(setting("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude")), "projects"), []string{"*/*.jsonl"}},
		{"codex", setting("CODEX_HOME", filepath.Join(home, ".codex")), []string{"sessions/*/*/*/rollout-*.jsonl", "archived_sessions/rollout-*.jsonl"}},
	} {
		if !directory(a.root) && !installed(a.name) {
			continue
		}
		inv.agents = append(inv.agents, a.name)
		// Expand only the fixed layout; never descend into artifact directories or linked stores.
		for _, p := range a.patterns {
			walkLayout(a.root, strings.Split(p, "/"), func(path string, entry os.DirEntry) {
				m := filenames[a.name].FindStringSubmatch(entry.Name())
				if m == nil {
					return
				}
				info, e := entry.Info()
				if e != nil || !info.Mode().IsRegular() {
					return
				}
				project, e := readProject(path, a.name, m[1])
				if e != nil {
					inv.warnings = append(inv.warnings, fmt.Sprintf("%s: %s: %v", a.name, entry.Name(), e))
				}
				inv.rows = append(inv.rows, session{a.name, path, m[1], project, info})
			}, &inv.warnings)
		}
	}
	data := setting("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	for _, x := range []struct{ name, cmd, dir string }{{"OpenCode", "opencode", filepath.Join(data, "opencode")}, {"Gemini CLI", "gemini", filepath.Join(home, ".gemini", "tmp")}, {"GitHub Copilot CLI", "copilot", filepath.Join(home, ".copilot", "session-state")}, {"Antigravity", "antigravity", filepath.Join(home, ".gemini", "antigravity")}, {"Cursor Agent", "cursor-agent", filepath.Join(home, ".cursor", "chats")}, {"Aider", "aider", ""}} {
		if installed(x.cmd) || directory(x.dir) {
			inv.other = append(inv.other, x.name)
		}
	}
	return inv
}
func walkLayout(dir string, parts []string, visit func(string, os.DirEntry), warnings *[]string) {
	if !filepath.IsAbs(dir) {
		*warnings = append(*warnings, "Ignoring relative store: "+dir)
		return
	}
	real, e := filepath.EvalSymlinks(dir)
	if errors.Is(e, os.ErrNotExist) {
		return
	}
	if e != nil {
		*warnings = append(*warnings, e.Error())
		return
	}
	if real != filepath.Clean(dir) {
		*warnings = append(*warnings, "Ignoring linked store: "+dir)
		return
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		*warnings = append(*warnings, e.Error())
		return
	}
	for _, v := range entries {
		if v.Type()&os.ModeSymlink != 0 {
			continue
		}
		ok, _ := filepath.Match(parts[0], v.Name())
		if !ok {
			continue
		}
		p := filepath.Join(dir, v.Name())
		if len(parts) == 1 {
			visit(p, v)
		} else if v.IsDir() {
			walkLayout(p, parts[1:], visit, warnings)
		}
	}
}
func readProject(path, agent, id string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, e := r.ReadSlice('\n')
		if errors.Is(e, bufio.ErrBufferFull) { // Large messages are common. Only parse bounded metadata records.
			var buf []byte
			for errors.Is(e, bufio.ErrBufferFull) {
				if len(buf) <= 1<<20 {
					buf = append(buf, line...)
				}
				line, e = r.ReadSlice('\n')
			}
			if len(buf) <= 1<<20 {
				line = append(buf, line...)
			} else {
				line = nil
			}
		}
		type meta struct{ ID, SessionID, Cwd string }
		var entry struct {
			meta
			Type    string
			Payload meta
		}
		if json.Unmarshal(line, &entry) == nil {
			m := entry.meta
			if agent == "codex" && entry.Type == "session_meta" {
				m = entry.Payload
			}
			key := m.ID
			if agent == "claude" {
				key = m.SessionID
			}
			if key == id && filepath.IsAbs(m.Cwd) {
				return filepath.Clean(m.Cwd), nil
			}
		}
		if e == io.EOF {
			return "", nil
		}
		if e != nil {
			return "", e
		}
	}
}
func projects(rows []session, agent string) []string {
	var names []string
	seen := map[string]bool{}
	for _, s := range rows {
		if s.agent == agent && !seen[s.project] {
			names = append(names, s.project)
			seen[s.project] = true
		}
	}
	slices.Sort(names)
	return names
}
