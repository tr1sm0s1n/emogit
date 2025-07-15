package models

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fileStatus struct {
	path   string
	staged bool
}

var (
	stagedStyle   = lipgloss.NewStyle().Bold(true)
	unstagedStyle = lipgloss.NewStyle().Faint(true)
)

func getUpdatedFiles() ([]fileStatus, error) {
	cmd := exec.Command(
		"git", "status",
		"--porcelain=v1",        // stable parse‑friendly format
		"--untracked-files=all", // list every untracked file
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	var files []fileStatus
	for line := range strings.SplitSeq(strings.TrimRight(out.String(), "\n"), "\n") {
		if len(line) < 3 {
			continue
		}
		x := line[:2] // XY status
		path := strings.TrimSpace(line[3:])

		// rename: "R  old -> new"
		if strings.Contains(path, " -> ") {
			if parts := strings.SplitN(path, " -> ", 2); len(parts) == 2 {
				path = parts[1]
			}
		}

		var staged bool
		if x == "??" {
			staged = false
		} else {
			staged = x[0] != ' '
		}

		files = append(files, fileStatus{
			path:   path,
			staged: staged,
		})
	}
	return files, nil
}

func toggleStage(f fileStatus) error {
	var cmd *exec.Cmd
	if f.staged {
		cmd = exec.Command("git", "reset", "--", f.path) // unstage
	} else {
		cmd = exec.Command("git", "add", "--", f.path) // stage
	}
	return cmd.Run()
}

func stageAll(files []fileStatus) error {
	if len(files) == 0 {
		return nil
	}
	args := []string{"add", "--"}
	for _, f := range files {
		if !f.staged {
			args = append(args, f.path)
		}
	}
	if len(args) == 2 { // nothing to stage
		return nil
	}
	return exec.Command("git", args...).Run()
}

type stageModel struct {
	files  []fileStatus
	cursor int
	err    error
}

func InitialStageModel() stageModel {
	files, err := getUpdatedFiles()
	return stageModel{files: files, err: err}
}

func (m stageModel) Init() tea.Cmd { return nil }

func (m stageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
		case " ", "enter":
			if len(m.files) == 0 {
				break
			}
			f := m.files[m.cursor]
			if err := toggleStage(f); err != nil {
				m.err = err
			} else {
				m.files, m.err = getUpdatedFiles()
				if m.cursor >= len(m.files) {
					m.cursor = len(m.files) - 1
				}
			}
		case "a":
			if err := stageAll(m.files); err != nil {
				m.err = err
			} else {
				m.files, m.err = getUpdatedFiles()
				m.cursor = 0
			}
		case "r":
			m.files, m.err = getUpdatedFiles()
			if m.cursor >= len(m.files) {
				m.cursor = len(m.files) - 1
			}
		case "right":
			n := initialCommitModel()
			return n, n.Init()
		}
	}
	return m, nil
}

func (m stageModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("error: %v\n", m.err)
	}
	if len(m.files) == 0 {
		return "No changes to show.\n"
	}

	var b strings.Builder
	b.WriteString("▲/▼: navigate, enter: toggle, a: stage all, r: refresh, ▶: next, esc: quit\n\n")

	for i, f := range m.files {
		line := f.path
		if f.staged {
			line = stagedStyle.Render("[S] " + line)
		} else {
			line = unstagedStyle.Render("[ ] " + line)
		}
		if i == m.cursor {
			line = cursorStyle.Render("> ") + line
		} else {
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}
