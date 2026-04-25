package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

const (
	maxVisible = 10
	maxWidth   = 120
)

var (
	styleHeader   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("4")).Bold(true)
	styleNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	styleMatch    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	styleHint     = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
)

type pickerModel struct {
	all       []string
	matches   fuzzy.Matches
	cursor    int
	scroll    int
	input     textinput.Model
	selected  string
	cancelled bool
	width     int
}

func newPickerModel(all []string, initial string) pickerModel {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Prompt = "❯ "
	ti.CharLimit = 512
	ti.Width = 80
	ti.SetValue(initial)
	ti.CursorEnd()
	ti.Focus()

	m := pickerModel{
		all:   all,
		input: ti,
		width: 80,
	}
	m.refilter()
	return m
}

func (m pickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *pickerModel) refilter() {
	q := strings.TrimSpace(m.input.Value())
	if q == "" {
		m.matches = make(fuzzy.Matches, len(m.all))
		for i, s := range m.all {
			m.matches[i] = fuzzy.Match{Str: s, Index: i}
		}
	} else {
		m.matches = fuzzy.Find(q, m.all)
	}
	m.cursor = 0
	m.scroll = 0
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		if m.width > maxWidth {
			m.width = maxWidth
		}
		m.input.Width = m.width - 4

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "ctrl+g":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			if len(m.matches) > 0 {
				m.selected = m.matches[m.cursor].Str
			} else {
				m.selected = m.input.Value()
			}
			return m, tea.Quit

		case "up", "ctrl+p", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scroll {
					m.scroll = m.cursor
				}
			}
			return m, nil

		case "down", "ctrl+n", "ctrl+j":
			if m.cursor < len(m.matches)-1 {
				m.cursor++
				if m.cursor >= m.scroll+maxVisible {
					m.scroll = m.cursor - maxVisible + 1
				}
			}
			return m, nil
		}
	}

	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prev {
		m.refilter()
	}
	return m, cmd
}

func (m pickerModel) View() string {
	var b strings.Builder

	count := len(m.matches)
	header := styleHeader.Render("history") + "  " +
		styleHint.Render("↑/↓ select · Enter accept · Esc cancel · ") +
		styleHeader.Render(itoa(count)+" matches")
	b.WriteString(header)
	b.WriteString("\n\n")

	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if count == 0 {
		b.WriteString(styleHint.Render("  (no matches)"))
		return b.String()
	}

	end := m.scroll + maxVisible
	if end > count {
		end = count
	}
	for i := m.scroll; i < end; i++ {
		row := m.matches[i].Str
		if i == m.cursor {
			b.WriteString(styleSelected.Render("❯ " + truncate(row, m.width-4)))
		} else {
			b.WriteString(styleNormal.Render("  " + truncate(row, m.width-4)))
		}
		b.WriteByte('\n')
	}

	if count > maxVisible {
		b.WriteString(styleHint.Render("  ... " + itoa(count-maxVisible) + " more"))
	}
	return b.String()
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if len(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return s[:w-1] + "…"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
