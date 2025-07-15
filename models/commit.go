package models

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/tr1sm0s1n/emogit/assets"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#DC143C"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#696969"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()

	focusedButton = focusedStyle.Render("[ Commit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Commit"))
)

type commitModel struct {
	width      int
	focusIndex int
	inputs     []textinput.Model
	cursorMode cursor.Mode
	message    string
}

func initialCommitModel() commitModel {
	m := commitModel{
		width:  100,
		inputs: make([]textinput.Model, 2),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32
		t.Width = 32

		switch i {
		case 0:
			t.Placeholder = "Emoji target"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "Commit message"
			t.CharLimit = 72
			t.Width = 72
		}

		m.inputs[i] = t
	}

	return m
}

func (m commitModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m commitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Change cursor mode
		case "ctrl+r":
			m.cursorMode++
			if m.cursorMode > cursor.CursorHide {
				m.cursorMode = cursor.CursorBlink
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := range m.inputs {
				cmds[i] = m.inputs[i].Cursor.SetMode(m.cursorMode)
			}
			return m, tea.Batch(cmds...)

		// Set focus to next input
		case "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the Commit button was focused?
			// If so, exit.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				i, _ := strconv.Atoi(m.inputs[0].Value())
				emo := strings.Split(assets.Emojis[i], " ")[0]
				m.message = fmt.Sprintf("%s | %s", emo, m.inputs[1].Value())
				m.Execute()
				return m, tea.Quit
			}

			// Cycle indexes
			if s == "up" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		case "shift+tab":
			n := InitialStageModel()
			return n, n.Init()
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *commitModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m commitModel) View() string {
	var b strings.Builder
	b.WriteString("▲/▼: navigate, enter: proceed, shift+tab: previous, esc: quit\n\n")
	b.WriteString("Targets:\n")

	lineLen := 0
	for i, v := range assets.Emojis {
		s := fmt.Sprintf("%d:[%s] ", i, v)
		strLen := runewidth.StringWidth(s) // ✅ handle emoji widths
		if lineLen+strLen > m.width {
			b.WriteRune('\n')
			lineLen = 0
		}

		b.WriteString(s)
		lineLen += strLen
	}
	b.WriteString("\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)
	return b.String()
}

func (m commitModel) Execute() {
	cmd := exec.Command("git", "commit", "-s", "-m", m.message)
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	cmd.Start()

	cmdLog := func(r io.Reader) {
		s := bufio.NewScanner(r)
		for s.Scan() {
			fmt.Print("\033[2K\r")
			fmt.Println(s.Text())
		}
	}

	go cmdLog(stdoutPipe)
	go cmdLog(stderrPipe)

	cmd.Wait()
}
