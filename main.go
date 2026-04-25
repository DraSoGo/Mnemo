package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Exit codes
//   0  command selected, printed to stdout
//   1  user cancelled (Esc / Ctrl+C)
//   2  no history available or read error
const (
	exitOK        = 0
	exitCancelled = 1
	exitError     = 2
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "predict":
			runPredict(os.Args[2:])
			return
		case "warmup":
			runWarmup(os.Args[2:])
			return
		}
	}

	initial := ""
	if len(os.Args) > 1 {
		if os.Args[1] == "pick" {
			initial = strings.Join(os.Args[2:], " ")
		} else {
			initial = strings.Join(os.Args[1:], " ")
		}
	}

	entries, err := LoadHistory("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo: cannot read history: %v\n", err)
		os.Exit(exitError)
	}
	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "mnemo: history is empty")
		os.Exit(exitError)
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo: cannot open /dev/tty: %v\n", err)
		os.Exit(exitError)
	}
	defer tty.Close()

	m := newPickerModel(entries, initial)
	p := tea.NewProgram(
		m,
		tea.WithInput(tty),
		tea.WithOutput(tty),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo: tui error: %v\n", err)
		os.Exit(exitError)
	}

	pm := finalModel.(pickerModel)
	if pm.cancelled || pm.selected == "" {
		os.Exit(exitCancelled)
	}
	fmt.Print(pm.selected)
	os.Exit(exitOK)
}
