package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tr1sm0s1n/emogit/models"
)

var (
	version = "v2.1.0 (Korryn)"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "v", "version":
			fmt.Println(version)
			return
		}
	}

	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(os.Stderr, "git is required but not found in PATH")
		os.Exit(1)
	}
	if _, err := exec.Command("git", "rev-parse", "--git-dir").Output(); err != nil {
		fmt.Fprintln(os.Stderr, "Not inside a Git repository")
		os.Exit(1)
	}

	p := tea.NewProgram(models.InitialStageModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running program:", err)
		os.Exit(1)
	}
}
