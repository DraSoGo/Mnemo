package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultModel     = "qwen2.5-coder:1.5b"
	defaultEndpoint  = "http://localhost:11434/api/generate"
	defaultTimeout   = 30 * time.Second
	defaultKeepAlive = "30m"
)

// runPredict reads a buffer from args, queries Ollama, prints prediction to stdout.
//
//	0 — prediction printed
//	1 — empty buffer / no prediction
//	2 — Ollama unreachable or error
func runPredict(args []string) {
	buffer := strings.TrimSpace(strings.Join(args, " "))
	if buffer == "" {
		os.Exit(1)
	}

	pred, err := predictOllama(buffer, ollamaConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "mnemo predict: %v\n", err)
		os.Exit(2)
	}
	if pred == "" {
		os.Exit(1)
	}
	fmt.Print(pred)
}

// runWarmup sends a tiny prompt so Ollama loads the model into RAM.
// Intended to be called in the background from the plugin so the first
// real prediction is fast.
func runWarmup(_ []string) {
	cfg := ollamaConfig()
	if _, err := predictOllama("ls", cfg); err != nil {
		fmt.Fprintf(os.Stderr, "mnemo warmup: %v\n", err)
		os.Exit(2)
	}
}

type ollamaCfg struct {
	model     string
	endpoint  string
	timeout   time.Duration
	keepAlive string
}

func ollamaConfig() ollamaCfg {
	cfg := ollamaCfg{
		model:     getenv("MNEMO_MODEL", defaultModel),
		endpoint:  getenv("MNEMO_OLLAMA_URL", defaultEndpoint),
		timeout:   defaultTimeout,
		keepAlive: getenv("MNEMO_KEEP_ALIVE", defaultKeepAlive),
	}
	if s := os.Getenv("MNEMO_TIMEOUT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			cfg.timeout = time.Duration(n) * time.Second
		}
	}
	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func predictOllama(buffer string, cfg ollamaCfg) (string, error) {
	// Raw completion mode bypasses the chat template — the model sees the
	// prompt verbatim and continues the text. This works far better than
	// chat-style instructions for a 1.5B coder model.
	prompt := "# Shell command completions. Output ONE complete command per line.\n" +
		"$ ls -la /tmp\n" +
		"$ git commit -m \"fix: typo in README\"\n" +
		"$ docker ps -a\n" +
		"$ " + buffer

	payload := map[string]any{
		"model":      cfg.model,
		"prompt":     prompt,
		"stream":     false,
		"raw":        true,
		"keep_alive": cfg.keepAlive,
		"options": map[string]any{
			"num_predict": 80,
			"temperature": 0.1,
			"top_p":       0.9,
			"stop":        []string{"\n", "$ ", "```", "#"},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}

	resp_line := result.Response
	if idx := strings.IndexByte(resp_line, '\n'); idx >= 0 {
		resp_line = resp_line[:idx]
	}
	// Model may echo buffer back; strip if so. Otherwise treat as continuation.
	resp_line = strings.TrimPrefix(resp_line, "$ ")
	resp_line = strings.TrimPrefix(resp_line, buffer)
	resp_line = strings.TrimRight(resp_line, " `\"'")
	if resp_line == "" {
		return "", nil
	}
	return buffer + resp_line, nil
}
