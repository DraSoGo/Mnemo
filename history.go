package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadHistory reads the zsh history file and returns entries newest-first,
// deduplicated (keeping the most recent occurrence of each command).
func LoadHistory(path string) ([]string, error) {
	if path == "" {
		path = defaultHistFile()
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var raw []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var pending strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		// Multi-line continuation: previous line ended with backslash.
		if pending.Len() > 0 {
			pending.WriteByte('\n')
			pending.WriteString(line)
			if !strings.HasSuffix(line, "\\") {
				raw = append(raw, stripExtended(pending.String()))
				pending.Reset()
			}
			continue
		}
		if strings.HasSuffix(line, "\\") {
			pending.WriteString(line)
			continue
		}
		raw = append(raw, stripExtended(line))
	}
	if pending.Len() > 0 {
		raw = append(raw, stripExtended(pending.String()))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Dedupe newest-first while reversing order.
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for i := len(raw) - 1; i >= 0; i-- {
		cmd := strings.TrimSpace(raw[i])
		if cmd == "" {
			continue
		}
		if _, dup := seen[cmd]; dup {
			continue
		}
		seen[cmd] = struct{}{}
		out = append(out, cmd)
	}
	return out, nil
}

// stripExtended strips zsh extended-history prefix `: timestamp:elapsed;`.
func stripExtended(line string) string {
	if !strings.HasPrefix(line, ": ") {
		return line
	}
	if idx := strings.IndexByte(line, ';'); idx >= 0 {
		return line[idx+1:]
	}
	return line
}

func defaultHistFile() string {
	if p := os.Getenv("HISTFILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".zsh_history")
}
