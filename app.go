package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type App struct {
	ctx      context.Context
	serveURL string
	servePID int
}

func NewApp() *App {
	return &App{
		serveURL: "http://localhost:11434",
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.ensureServe()
}

func (a *App) shutdown(ctx context.Context) {
	a.stopServe()
}

// ensureServe starts ai serve --daemon if not already running.
func (a *App) ensureServe() {
	if a.isServeRunning() {
		return
	}

	exe := a.findAI()
	if exe == "" {
		return
	}

	cmd := exec.Command(exe, "serve", "--daemon", "--port", "11434")
	cmd.Start()

	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if a.isServeRunning() {
			break
		}
	}
}

func (a *App) stopServe() {
	pidPath := a.pidPath()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	if proc, err := os.FindProcess(pid); err == nil {
		proc.Kill()
	}
	os.Remove(pidPath)
}

func (a *App) isServeRunning() bool {
	resp, err := http.Get(a.serveURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (a *App) findAI() string {
	candidates := []string{
		"/opt/homebrew/bin/ai",
		"/usr/local/bin/ai",
	}
	home, _ := os.UserHomeDir()
	candidates = append(candidates, filepath.Join(home, "go", "bin", "ai"))
	if p, err := exec.LookPath("ai"); err == nil {
		return p
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func (a *App) pidPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ai", "serve.pid")
}

// --- Exposed to frontend ---

type ServerStatus struct {
	Running bool   `json:"running"`
	Model   string `json:"model"`
	GPU     bool   `json:"gpu"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

func (a *App) GetStatus() ServerStatus {
	resp, err := http.Get(a.serveURL + "/health")
	if err != nil {
		return ServerStatus{URL: a.serveURL}
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&health)

	model, _ := health["model"].(string)
	gpu, _ := health["gpu"].(bool)
	version, _ := health["version"].(string)

	return ServerStatus{
		Running: true,
		Model:   model,
		GPU:     gpu,
		Version: version,
		URL:     a.serveURL,
	}
}

func (a *App) ListModels() []string {
	resp, err := http.Get(a.serveURL + "/v1/models")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	var models []string
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models
}

func (a *App) LoadModel(name string) error {
	exe := a.findAI()
	if exe == "" {
		return fmt.Errorf("ai binary not found")
	}
	a.stopServe()
	time.Sleep(500 * time.Millisecond)

	cmd := exec.Command(exe, "serve", fmt.Sprintf("model=%s", name), "--daemon", "--port", "11434")
	if err := cmd.Start(); err != nil {
		return err
	}

	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		if a.isServeRunning() {
			return nil
		}
	}
	return fmt.Errorf("serve did not start in time")
}

func (a *App) SendMessage(message string) string {
	body := map[string]interface{}{
		"model": "",
		"messages": []map[string]string{
			{"role": "user", "content": message},
		},
		"stream": false,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(a.serveURL+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Sprintf("[error: %v]", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Sprintf("[parse error: %v]", err)
	}
	if len(result.Choices) == 0 {
		return "[no response]"
	}
	return result.Choices[0].Message.Content
}

func (a *App) PullModel(name string) string {
	exe := a.findAI()
	if exe == "" {
		return "ai binary not found"
	}
	cmd := exec.Command(exe, "pull", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("error: %v\n%s", err, string(out))
	}
	return string(out)
}
