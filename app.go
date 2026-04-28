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
	"runtime"
	"strconv"
	"strings"
	"time"

)

type App struct {
	ctx      context.Context
	serveURL string
	servePID int
	pullCmd  *exec.Cmd
}

const defaultServePort = "11435"

func NewApp() *App {
	return &App{
		serveURL: "http://127.0.0.1:" + defaultServePort,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.ensureAI()
	a.ensureServe()
	go func() {
		time.Sleep(3 * time.Second)
		setupTray(a, trayIconPNG)
	}()
}

func (a *App) shutdown(ctx context.Context) {
	teardownTray()
}

// ensureServe checks if ai serve is already running. Only starts a new one
// if nothing is listening on the port.
func (a *App) ensureServe() {
	if a.isServeRunning() {
		return
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
	// Check our own install dir first
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	ours := filepath.Join(installDir(), "ai"+ext)
	if _, err := os.Stat(ours); err == nil {
		return ours
	}

	// Then PATH
	if p, err := exec.LookPath("ai"); err == nil {
		return p
	}

	// Common locations
	candidates := []string{
		"/opt/homebrew/bin/ai",
		"/usr/local/bin/ai",
	}
	home, _ := os.UserHomeDir()
	candidates = append(candidates, filepath.Join(home, "go", "bin", "ai"+ext))
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

func (a *App) GetSetupStatus() string {
	exe := a.findAI()
	if exe == "" {
		return "not installed"
	}
	cmd := exec.Command(exe, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "installed (unknown version)"
	}
	return strings.TrimSpace(string(out))
}

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

	cmd := exec.Command(exe, "serve", fmt.Sprintf("model=%s", name), "--daemon", "--port", defaultServePort)
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
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant. Answer concisely."},
			{"role": "user", "content": message},
		},
		"max_tokens": 200,
		"stream":     false,
	}
	jsonBody, _ := json.Marshal(body)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(a.serveURL+"/v1/chat/completions", "application/json", bytes.NewReader(jsonBody))
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
	return a.RunCommand("pull " + name)
}

func (a *App) RunCommand(input string) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return a.commandHelp()
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "help":
		return a.commandHelp()
	case "pull":
		if len(args) == 0 {
			return "usage: /pull <org/model>"
		}
		return a.runAI("pull", args...)
	case "models":
		return a.runAI("models")
	case "info":
		if len(args) == 0 {
			return "usage: /info <model>"
		}
		return a.runAI("info", args[0])
	case "load":
		if len(args) == 0 {
			return "usage: /load <model>"
		}
		if err := a.LoadModel(args[0]); err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return fmt.Sprintf("loaded %s", args[0])
	case "gpus":
		return a.runAI("gpus")
	case "bench":
		return a.runAI("bench")
	case "quantize":
		if len(args) < 1 {
			return "usage: /quantize <model> [q8|q4|f16]"
		}
		return a.runAI("quantize", args...)
	case "train":
		if len(args) < 1 {
			return "usage: /train data=<file> [dim=N] [steps=N]"
		}
		return a.runAI("train", args...)
	case "serve":
		return a.runAI("serve", args...)
	case "status":
		s := a.GetStatus()
		if !s.Running {
			return "serve: not running"
		}
		gpu := ""
		if s.GPU {
			gpu = " (GPU)"
		}
		return fmt.Sprintf("serve: %s%s\n%s", s.Model, gpu, s.Version)
	case "stop":
		a.stopServe()
		return "serve stopped"
	default:
		return fmt.Sprintf("unknown command: /%s\n\n%s", cmd, a.commandHelp())
	}
}

func (a *App) PullModelWithProgress(name string) string {
	exe := a.findAI()
	if exe == "" {
		return "error: ai binary not found"
	}

	a.pullCmd = exec.Command(exe, "pull", name)
	stdout, err := a.pullCmd.StdoutPipe()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	a.pullCmd.Stderr = a.pullCmd.Stdout

	if err := a.pullCmd.Start(); err != nil {
		a.pullCmd = nil
		return fmt.Sprintf("error: %v", err)
	}

	var output strings.Builder
	buf := make([]byte, 256)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	a.pullCmd.Wait()
	a.pullCmd = nil
	return strings.TrimSpace(output.String())
}

func (a *App) CancelPull() string {
	if a.pullCmd != nil && a.pullCmd.Process != nil {
		a.pullCmd.Process.Kill()
		a.pullCmd = nil
		return "download cancelled"
	}
	return "no download in progress"
}

func (a *App) commandHelp() string {
	return `/help                Show this help
/pull <org/model>    Download model from HuggingFace
/models              List downloaded models
/info <model>        Show model architecture
/load <model>        Load model into serve
/status              Show serve status
/stop                Stop serve daemon
/gpus                Detect hardware
/bench               GPU benchmark
/quantize <model>    Quantize model
/train data=<file>   Train a model`
}

func (a *App) runAI(command string, args ...string) string {
	exe := a.findAI()
	if exe == "" {
		return "ai binary not found — install with: brew install tensorwire/tap/ai"
	}
	allArgs := append([]string{command}, args...)
	cmd := exec.Command(exe, allArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("%s\nerror: %v", string(out), err)
	}
	return strings.TrimSpace(string(out))
}
