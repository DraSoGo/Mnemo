package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripExtended(t *testing.T) {
	cases := map[string]string{
		": 1700000000:0;ls -la":      "ls -la",
		": 1700000000:5;cd ~/repo":   "cd ~/repo",
		"plain command":              "plain command",
		":not_extended":              ":not_extended",
		": 1700000000:0;":            "",
		":  1234:1;echo with;semis": "echo with;semis",
	}
	for in, want := range cases {
		got := stripExtended(in)
		if got != want {
			t.Errorf("stripExtended(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadHistoryDedupAndOrder(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "zsh_history")
	content := ": 1:0;ls\n" +
		": 2:0;cd /tmp\n" +
		": 3:0;ls\n" +
		": 4:0;echo hi\n" +
		": 5:0;cd /tmp\n"
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadHistory(f)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"cd /tmp", "echo hi", "ls"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
