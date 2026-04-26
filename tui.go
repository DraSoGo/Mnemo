package main

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

const (
	defaultMaxVisible = 10
	maxWidth          = 120
)

var (
	styleHeader   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("4")).Bold(true)
	styleNormal   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	styleMatch    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	styleHint     = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
)

type pickerModel struct {
	all        []string
	matches    fuzzy.Matches
	cursor     int
	scroll     int
	input      textinput.Model
	selected   string
	cancelled  bool
	width      int
	maxVisible int
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
		all:        all,
		input:      ti,
		width:      80,
		maxVisible: defaultMaxVisible,
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
		// header(1) + blank(1) + input(1) + blank(1) + hint(1) = 5 lines overhead
		if visible := msg.Height - 5; visible >= 3 {
			m.maxVisible = visible
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "ctrl+f":
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
				if m.cursor >= m.scroll+m.maxVisible {
					m.scroll = m.cursor - m.maxVisible + 1
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
		styleHeader.Render(strconv.Itoa(count)+" matches")
	b.WriteString(header)
	b.WriteString("\n\n")

	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if count == 0 {
		b.WriteString(styleHint.Render("  (no matches)"))
		return b.String()
	}

	end := m.scroll + m.maxVisible
	if end > count {
		end = count
	}
	for i := m.scroll; i < end; i++ {
		if i == m.cursor {
			b.WriteString(styleSelected.Render("❯ " + truncate(m.matches[i].Str, m.width-4)))
		} else {
			b.WriteString("  " + renderHighlighted(m.matches[i], m.width-4))
		}
		b.WriteByte('\n')
	}

	if count > m.maxVisible {
		b.WriteString(styleHint.Render("  ... " + strconv.Itoa(count-m.maxVisible) + " more"))
	}
	return b.String()
}

// renderHighlighted renders a fuzzy match with matched characters highlighted.
// Non-selected rows only — selected rows use a block style instead.
func renderHighlighted(m fuzzy.Match, maxW int) string {
	runes := []rune(m.Str)

	// Truncate rune slice first so indices stay valid.
	suffix := ""
	if maxW > 0 && len(runes) > maxW {
		if maxW <= 1 {
			return styleNormal.Render("…")
		}
		runes = runes[:maxW-1]
		suffix = "…"
	}

	if len(m.MatchedIndexes) == 0 {
		return styleNormal.Render(string(runes) + suffix)
	}

	// Build a set of matched rune indices for O(1) lookup.
	idxSet := make(map[int]bool, len(m.MatchedIndexes))
	for _, idx := range m.MatchedIndexes {
		idxSet[idx] = true
	}

	var b strings.Builder
	for i, r := range runes {
		if idxSet[i] {
			b.WriteString(styleMatch.Render(string(r)))
		} else {
			b.WriteString(styleNormal.Render(string(r)))
		}
	}
	b.WriteString(styleNormal.Render(suffix))
	return b.String()
}

// truncate shortens s to at most w runes, appending "…" if truncated.
// Uses rune count to correctly handle multi-byte Unicode characters.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return string(runes[:w-1]) + "…"
}
