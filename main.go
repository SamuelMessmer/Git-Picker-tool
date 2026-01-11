package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Types ---

type fileStatus struct {
	status string
	path   string
}

type model struct {
	files   []fileStatus
	cursor  int
	err     error
	addMode bool
	gitRoot string
}

type execFinishedMsg struct{ err error }

// --- Git Helpers ---

func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

func getGitStatus(gitRoot string) []fileStatus {
	cmd := exec.Command("git", "status", "--porcelain")
	if gitRoot != "" {
		cmd.Dir = gitRoot
	}

	out, err := cmd.Output()
	if err != nil {
		return []fileStatus{}
	}

	lines := strings.Split(string(out), "\n")
	var files []fileStatus
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		status := line[:2]
		path := line[3:]
		files = append(files, fileStatus{status: status, path: path})
	}

	// Sort priorities:
	sort.SliceStable(files, func(i, j int) bool {
		s1 := files[i].status
		s2 := files[j].status

		getPriority := func(s string) int {
			if len(s) < 2 {
				return 5
			}
			index := s[0]
			worktree := s[1]

			if index == '?' {
				return 0 // Top Priority: Untracked
			}
			if index != ' ' && worktree != ' ' {
				return 1 // Partially Staged
			}
			if index != ' ' {
				return 2 // Fully Staged
			}
			return 3 // Unstaged
		}

		p1 := getPriority(s1)
		p2 := getPriority(s2)

		if p1 != p2 {
			return p1 < p2
		}
		return files[i].path < files[j].path
	})

	return files
}

// --- Bubble Tea Logic ---

func initialModel(addMode bool) model {
	gitRoot, err := getGitRoot()
	m := model{
		addMode: addMode,
		gitRoot: gitRoot,
	}

	if err != nil {
		m.err = err
		return m
	}

	m.files = getGitStatus(gitRoot)
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.files) == 0 {
				return m, nil
			}
			selectedFile := m.files[m.cursor]
			action := "diff"
			if m.addMode {
				action = "add"
			}
			return m, openGitCommand(selectedFile, action, m.gitRoot)
		
		case "u":
			if len(m.files) == 0 {
				return m, nil
			}
			selectedFile := m.files[m.cursor]
			return m, openGitCommand(selectedFile, "unstage", m.gitRoot)

		case "a":
			m.addMode = !m.addMode
		}

	case execFinishedMsg:
		m.files = getGitStatus(m.gitRoot)
		if len(m.files) == 0 {
			return m, tea.Quit
		}
		if m.cursor >= len(m.files) {
			m.cursor = len(m.files) - 1
		}
		return m, nil
	}
	return m, nil
}

func openGitCommand(file fileStatus, action string, gitRoot string) tea.Cmd {
	var args []string

	switch action {
	case "add":
		args = []string{"add", file.path}
	
	case "unstage":
		// FIX: Unterscheidung zwischen "Neu hinzugefügt" und "Modifiziert"
		// Wenn Status mit 'A' beginnt (Added), müssen wir 'rm --cached' nutzen.
		// Bei 'M' (Modified) oder 'D' (Deleted) nutzen wir 'restore --staged'.
		if strings.HasPrefix(file.status, "A") {
			args = []string{"rm", "--cached", file.path}
		} else {
			args = []string{"restore", "--staged", file.path}
		}

	default: // "diff"
		if strings.HasPrefix(file.status, "?") {
			// Untracked files -> Zeige Content als Diff (via /dev/null trick)
			// Kein --no-pager, damit der Standard Pager (less) aufgeht
			args = []string{"diff", "--no-index", "/dev/null", file.path}
		} else {
			isStaged := file.status[0] != ' ' && file.status[1] == ' '
			
			args = []string{"diff"}
			if isStaged {
				args = append(args, "--cached")
			}
			args = append(args, "--color=always", file.path)
		}
	}

	c := exec.Command("git", args...)

	if gitRoot != "" {
		c.Dir = gitRoot
	}

	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execFinishedMsg{err: err}
	})
}

// --- UI Styling & View ---

var (
	cursorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // 🩷 Hot Pink 
	addedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // 💚 Spring Green 
	modifiedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // 🧡 Orange / Gold 
	deletedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("160")) // 🔴 Rot 
	untrackedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // 🔵 Deep Sky Blue 
	defaultStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")) // ⚪ Weiß 
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // 🚨 Knallrot 
)

func getStatusDisplay(status string) (string, lipgloss.Style) {
	if len(status) < 2 {
		return status, defaultStyle
	}
	index := status[0]
	worktree := status[1]

	if index == '?' {
		return "❓ " + status, untrackedStyle
	}
	if index != ' ' && worktree == ' ' {
		return "✅ " + status, addedStyle
	}
	if worktree == 'M' {
		return "📝 " + status, modifiedStyle
	}
	if worktree == 'D' {
		return "🗑️ " + status, deletedStyle
	}
	if index != ' ' && worktree != ' ' {
		return "⚠️ " + status, modifiedStyle
	}

	return "  " + status, defaultStyle
}

func (m model) View() string {
	if m.err != nil {
		s := errorStyle.Render("Error: ") + m.err.Error() + "\n\n"
		s += "Please run this command from within a git repository.\n"
		s += "\nPress q to quit.\n"
		return s
	}

	s := "GitPicker"

	if m.gitRoot != "" {
		cwd, err := os.Getwd()
		if err == nil {
			relPath, err := filepath.Rel(m.gitRoot, cwd)
			if err == nil && relPath != "." {
				s += fmt.Sprintf(" (in %s)", relPath)
			}
		}
	}

	s += "\n\n"

	if len(m.files) == 0 {
		s += "No changes found.\n"
		s += "\nPress q to quit.\n"
		return s
	}

	for i, file := range m.files {
		statusIcon, style := getStatusDisplay(file.status)
		statusStr := style.Render(fmt.Sprintf("[%s]", statusIcon))

		if m.cursor == i {
			line := fmt.Sprintf("%s %s %s", cursorStyle.Render(">"), statusStr, cursorStyle.Render(file.path))
			s += "  " + line + "\n"
		} else {
			s += fmt.Sprintf("    %s %s\n", statusStr, file.path)
		}
	}

	s += "\nPress q to quit. Enter to "
	if m.addMode {
		s += modifiedStyle.Render("ADD")
	} else {
		s += untrackedStyle.Render("DIFF")
	}
	s += ". 'a' to toggle mode. 'u' to unstage.\n"
	return s
}

func main() {
	addMode := flag.Bool("a", false, "Start in add mode")
	flag.Parse()

	if !isGitRepo() {
		fmt.Fprintln(os.Stderr, errorStyle.Render("Error:")+" Not a git repository")
		os.Exit(1)
	}
	// INFO: "tea.WithAltScreen()" wurde entfernt.
	// Das Programm läuft jetzt inline im Terminal.
	// Git Diff öffnet seinen eigenen Pager (AltScreen), und beim Schließen (q)
	// bist du wieder hier im Inline-Menü.
	p := tea.NewProgram(
		initialModel(*addMode),
		// tea.WithAltScreen(), // <-- ENTFERNT
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
