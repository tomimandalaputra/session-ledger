package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

func printable(s string) string { q := strconv.Quote(s); return q[1 : len(q)-1] }
func label(p string) string {
	if p == "" {
		return "(unverified project)"
	}
	return printable(p)
}
func preview(inv inventory) {
	fmt.Println("Session Ledger | Global preview\nRead-only. No files were moved.")
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 3, ' ', 0)
	fmt.Fprintln(w, "\nAGENT\tCAN MOVE\tKEEP\tPROJECT")
	var notes []string
	totalOK := 0
	for _, a := range inv.agents {
		ok, e := eligible(a, inv.rows)
		if e != nil {
			notes = append(notes, a+": "+e.Error())
		}
		counts := map[string][2]int{}
		for _, s := range inv.rows {
			if s.agent != a {
				continue
			}
			c := counts[s.project]
			if s.project != "" && ok[s.id] {
				c[0]++
				totalOK++
			} else {
				c[1]++
			}
			counts[s.project] = c
		}
		if len(counts) == 0 {
			fmt.Fprintf(w, "%s\t0\t0\t(no local sessions)\n", a)
		}
		for _, p := range projects(inv.rows, a) {
			c := counts[p]
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", a, c[0], c[1], label(p))
		}
	}
	w.Flush()
	fmt.Printf("\n%d sessions · %d eligible · %d protected or unverified\n", len(inv.rows), totalOK, len(inv.rows)-totalOK)
	for _, n := range append(notes, inv.warnings...) {
		fmt.Println("Note:", printable(n))
	}
	if len(inv.other) > 0 {
		fmt.Println("Detected only (not counted):", strings.Join(inv.other, ", "))
	}
	if totalOK > 0 {
		fmt.Println("For interactive cleanup: session-ledger --clean")
	}
}

type menu struct {
	inv                       inventory
	agents, paths             []string
	agent                     string
	marked                    map[int]bool
	stage, pos, height, width int
	selected                  []session
	confirmed                 bool
}

func (m menu) Init() tea.Cmd { return nil }
func (m menu) count() int {
	switch m.stage {
	case 0:
		return len(m.agents)
	case 1:
		return len(m.paths) + 1
	default:
		return 2
	}
}
func (m menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch k := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = k.Height
		m.width = k.Width
	case tea.KeyMsg:
		switch k.String() {
		case "esc", "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.pos = (m.pos + m.count() - 1) % m.count()
		case "down", "j":
			m.pos = (m.pos + 1) % m.count()
		case "left", "backspace":
			if m.stage > 0 {
				m.stage--
				m.pos = 0
			}
		case " ":
			if m.stage == 1 {
				if m.marked[m.pos] {
					delete(m.marked, m.pos)
				} else {
					if m.pos == len(m.paths) {
						clear(m.marked)
					} else {
						delete(m.marked, len(m.paths))
					}
					m.marked[m.pos] = true
				}
			}
		case "enter", "right":
			switch m.stage {
			case 0:
				m.agent = m.agents[m.pos]
				m.paths = projects(m.inv.rows, m.agent)
				m.marked = map[int]bool{}
			case 1:
				sel := m.marked
				if len(sel) == 0 {
					sel = map[int]bool{m.pos: true}
				}
				m.selected = nil
				chosen := map[string]bool{}
				for i := range sel {
					if i < len(m.paths) {
						chosen[m.paths[i]] = true
					}
				}
				for _, r := range m.inv.rows {
					if r.agent == m.agent && (sel[len(m.paths)] || chosen[r.project]) {
						m.selected = append(m.selected, r)
					}
				}
			case 2:
				m.confirmed = m.pos == 1
				return m, tea.Quit
			}
			m.stage++
			m.pos = 0
		}
	}
	return m, nil
}
func (m menu) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Session Ledger | Cleanup (%d/3)\n", m.stage+1)
	titles := []string{"Select an agent", "Select projects | " + m.agent, fmt.Sprintf("Check %d selected sessions and move eligible transcripts to Trash?", len(m.selected))}
	fmt.Fprintln(&b, titles[m.stage])
	n := max(1, m.height-5)
	start := max(0, m.pos-n+1)
	for i := start; i < min(start+n, m.count()); i++ {
		var text string
		switch m.stage {
		case 0:
			text = m.agents[i]
		case 1:
			text = "All projects"
			if i < len(m.paths) {
				text = label(m.paths[i])
			}
			if m.marked[i] {
				text = "[x] " + text
			} else {
				text = "[ ] " + text
			}
		case 2:
			text = []string{"No, cancel", "Yes, move to Trash"}[i]
		}
		prefix := "  "
		if i == m.pos {
			prefix = "> "
		}
		fmt.Fprintln(&b, ansi.Truncate(prefix+text, max(5, m.width-1), "…"))
	}
	hint := "↑↓ Move · Enter Select"
	if m.stage == 1 {
		hint += " · Space Toggle"
	}
	if m.stage > 0 {
		hint += " · ← Back"
	}
	return b.String() + "\n" + hint + " · Esc Cancel"
}
func run() error {
	clean := flag.Bool("clean", false, "select projects and confirm moving eligible transcripts to Trash")
	show := flag.Bool("preview", false, "show all local sessions without changing files (default)")
	ver := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *ver {
		if flag.NArg() != 0 || *clean || *show {
			return fmt.Errorf("--version cannot be combined with other options; see --help")
		}
		fmt.Println(versionLine())
		return nil
	}
	if flag.NArg() != 0 || (*clean && *show) {
		return fmt.Errorf("use --preview or --clean; see --help")
	}
	home, e := os.UserHomeDir()
	if e != nil {
		return e
	}
	inv := scan(home)
	if !*clean {
		preview(inv)
		return nil
	}
	for _, w := range inv.warnings {
		fmt.Fprintln(os.Stderr, "Note:", printable(w))
	}
	var agents []string
	for _, a := range inv.agents {
		if len(projects(inv.rows, a)) > 0 {
			agents = append(agents, a)
		}
	}
	if len(agents) == 0 {
		fmt.Println("No local sessions found.")
		return nil
	}
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return fmt.Errorf("interactive cleanup requires a terminal")
	}
	result, e := tea.NewProgram(menu{inv: inv, agents: agents, width: 80, height: 24}, tea.WithAltScreen()).Run()
	if e != nil {
		return e
	}
	m := result.(menu)
	if !m.confirmed {
		fmt.Println("Cancelled. No sessions were moved.")
		return nil
	}
	n, k, e := cleanup(m.selected)
	fmt.Printf("Moved: %d. Not moved: %d.\n", n, k)
	return e
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "Session Ledger:", printable(e.Error()))
		os.Exit(1)
	}
}
