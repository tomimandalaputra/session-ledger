package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func output(cmd string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, cmd, args...).Output()
}

// Read process arguments as well as executable names: npm launchers can run as node.
func busy(agent string) (bool, error) {
	var b []byte
	var e error
	if runtime.GOOS == "windows" {
		b, e = output("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", `$ErrorActionPreference='Stop'; Get-CimInstance Win32_Process | ForEach-Object { $_.Name; $_.CommandLine }`)
	} else {
		b, e = output("ps", "-Aww", "-o", "comm=", "-o", "args=")
	}
	if e != nil || len(b) == 0 {
		return true, fmt.Errorf("cannot verify running clients")
	}
	for _, line := range strings.Split(strings.ToLower(string(b)), "\n") {
		for _, word := range strings.Fields(strings.ReplaceAll(line, `\`, "/")) {
			word = strings.Trim(word, `"'`)
			for _, part := range strings.Split(word, "/") {
				switch part {
				case agent, agent + ".exe", agent + ".cmd", agent + ".js", agent + "-code":
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func eligible(agent string, rows []session) (map[string]bool, error) {
	ok := map[string]bool{}
	var ids []string
	for _, r := range rows {
		if r.agent == agent && r.project != "" {
			ids = append(ids, r.id)
			ok[r.id] = true
		}
	}
	if len(ids) == 0 {
		return ok, nil
	}
	if agent != "omp" {
		b, e := busy(agent)
		if e != nil {
			return nil, e
		}
		if b {
			return nil, fmt.Errorf("close %s before cleanup", agent)
		}
		return ok, nil
	}
	for len(ids) > 0 {
		n := min(len(ids), 100)
		b, e := output("omp", append([]string{"session", "lease", "--json", "--"}, ids[:n]...)...)
		if e != nil {
			return nil, fmt.Errorf("lease check unavailable: %w", e)
		}
		var v struct{ Held, Unknown []string }
		if json.Unmarshal(b, &v) != nil || v.Held == nil || v.Unknown == nil {
			return nil, errors.New("unrecognized lease response")
		}
		for _, id := range v.Held {
			delete(ok, id)
		}
		for _, id := range v.Unknown {
			delete(ok, id)
		}
		ids = ids[n:]
	}
	return ok, nil
}

func unchanged(s session) bool {
	real, e := filepath.EvalSymlinks(s.path)
	if e != nil || real != s.path {
		return false
	}
	now, e := os.Lstat(s.path)
	return e == nil && now.Mode().IsRegular() && os.SameFile(s.info, now) && now.Size() == s.info.Size() && now.ModTime().Equal(s.info.ModTime())
}
func trash(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("gio", "trash", "--", path)
	case "darwin":
		cmd = exec.Command("osascript", "-l", "JavaScript", "-e", `ObjC.import('Foundation'); function run(args) { var err=Ref(); if (!$.NSFileManager.defaultManager.trashItemAtURLResultingItemURLError($.NSURL.fileURLWithPath(args[0]), null, err)) throw Error(ObjC.unwrap(err[0].localizedDescription)); }`, "--", path)
	case "windows": // Move into the Shell's Recycle Bin namespace; never invoke a delete verb.
		cmd = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", `$ErrorActionPreference='Stop'; $p=$env:SESSION_LEDGER_ITEM; $bin=(New-Object -ComObject Shell.Application).Namespace(10); $bin.MoveHere($p,0); for($i=0;$i -lt 50 -and (Test-Path -LiteralPath $p);$i++){Start-Sleep -Milliseconds 100}; if(Test-Path -LiteralPath $p){throw 'Recycle Bin did not accept this item'}`)
		cmd.Env = append(os.Environ(), "SESSION_LEDGER_ITEM="+path)
	default:
		return errors.New("native Trash unavailable")
	}
	if b, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("native Trash: %w (%q)", e, strings.TrimSpace(string(b)))
	}
	if _, e := os.Lstat(path); !errors.Is(e, os.ErrNotExist) {
		return errors.New("native Trash did not move this item")
	}
	return nil
}
// movable keeps only the sessions that pass the eligibility check, so the cleanup menu never
// offers a project that cannot be moved. The notes say why sessions were left out.
func movable(inv inventory) (inventory, []string) {
	var notes []string
	var rows []session
	for _, a := range inv.agents {
		ok, e := eligible(a, inv.rows)
		if e != nil {
			notes = append(notes, a+": "+e.Error())
			continue
		}
		left := 0
		for _, r := range inv.rows {
			switch {
			case r.agent != a:
			case r.project != "" && ok[r.id]:
				rows = append(rows, r)
			default:
				left++
			}
		}
		if left > 0 {
			notes = append(notes, fmt.Sprintf("%s: %d in use or not verified, not listed", a, left))
		}
	}
	inv.rows = rows
	return inv, notes
}

const (
	whyProject = "project not verified"
	whyBusy    = "in use or not verified by the agent"
	whyChanged = "changed since the scan"
)

// cleanup moves the eligible rows to Trash. why counts the skipped rows by reason.
func cleanup(rows []session) (moved, kept int, why map[string]int, err error) {
	why = map[string]int{}
	if len(rows) == 0 {
		return
	}
	ok, e := eligible(rows[0].agent, rows)
	if e != nil {
		return 0, len(rows), why, e
	}
	for _, r := range rows {
		switch {
		case r.project == "":
			why[whyProject]++
		case !ok[r.id]:
			why[whyBusy]++
		case !unchanged(r):
			why[whyChanged]++
		default:
			if e := trash(r.path); e != nil {
				return moved, len(rows) - moved, why, fmt.Errorf("%s: %w", filepath.Base(r.path), e)
			}
			moved++
			continue
		}
		kept++
	}
	return
}
