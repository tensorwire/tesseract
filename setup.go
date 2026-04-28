package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	aiRepo       = "tensorwire/ai"
	aiReleaseAPI = "https://api.github.com/repos/" + aiRepo + "/releases/latest"
)

func (a *App) ensureAI() string {
	if exe := a.findAI(); exe != "" {
		return exe
	}

	log.Printf("[setup] ai binary not found, downloading...")
	exe, err := downloadAI()
	if err != nil {
		log.Printf("[setup] download failed: %v", err)
		return ""
	}
	log.Printf("[setup] installed: %s", exe)
	return exe
}

func installDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ai", "bin")
}

func downloadAI() (string, error) {
	suffix := platformSuffix()
	if suffix == "" {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	assetURL, err := findReleaseAsset(suffix)
	if err != nil {
		return "", err
	}

	dir := installDir()
	os.MkdirAll(dir, 0755)

	resp, err := http.Get(assetURL)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	if strings.HasSuffix(assetURL, ".zip") {
		return extractZip(resp.Body, dir, suffix)
	}
	return extractTarGz(resp.Body, dir)
}

func platformSuffix() string {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		return "darwin-arm64"
	case "darwin/amd64":
		return "darwin-amd64"
	case "linux/amd64":
		return "linux-amd64"
	case "linux/arm64":
		return "linux-arm64"
	case "windows/amd64":
		return "windows-amd64"
	default:
		return ""
	}
}

func findReleaseAsset(suffix string) (string, error) {
	resp, err := http.Get(aiReleaseAPI)
	if err != nil {
		return "", fmt.Errorf("github API: %w", err)
	}
	defer resp.Body.Close()

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("parse release: %w", err)
	}

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, suffix) {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no asset found for %s", suffix)
}

func extractTarGz(r io.Reader, dir string) (string, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var exePath string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		name := filepath.Base(hdr.Name)
		dst := filepath.Join(dir, name)

		f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)|0755)
		if err != nil {
			return "", err
		}
		io.Copy(f, tr)
		f.Close()

		if strings.HasPrefix(name, "ai-") || name == "ai" {
			exePath = dst
		}
	}

	if exePath != "" {
		// Rename to just "ai"
		finalPath := filepath.Join(dir, "ai")
		if exePath != finalPath {
			os.Rename(exePath, finalPath)
			exePath = finalPath
		}
	}
	return exePath, nil
}

func extractZip(r io.Reader, dir, suffix string) (string, error) {
	// Download to temp file first (zip needs seeking)
	tmp, err := os.CreateTemp("", "ai-*.zip")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := io.Copy(tmp, r); err != nil {
		return "", err
	}

	zr, err := zip.OpenReader(tmp.Name())
	if err != nil {
		return "", err
	}
	defer zr.Close()

	var exePath string
	for _, f := range zr.File {
		name := filepath.Base(f.Name)
		dst := filepath.Join(dir, name)

		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return "", err
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()

		if strings.HasPrefix(name, "ai-") || name == "ai" || strings.HasSuffix(name, ".exe") {
			exePath = dst
		}
	}

	if exePath != "" {
		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".exe"
		}
		finalPath := filepath.Join(dir, "ai"+ext)
		if exePath != finalPath {
			os.Rename(exePath, finalPath)
			exePath = finalPath
		}
	}
	return exePath, nil
}
