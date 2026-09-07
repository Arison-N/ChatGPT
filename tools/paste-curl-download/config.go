package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type appConfig struct {
	DestDir string `json:"destDir"`
}

var (
	configMu          sync.Mutex
	configPathForTest string
)

func configFilePath() string {
	if configPathForTest != "" {
		return configPathForTest
	}
	if runtime.GOOS == "windows" {
		if d := os.Getenv("APPDATA"); d != "" {
			return filepath.Join(d, "Zoom-loader", "config.json")
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "zoom-loader-config.json")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "Zoom-loader", "config.json")
	}
	return filepath.Join(home, ".config", "zoom-loader", "config.json")
}

func loadConfig() appConfig {
	configMu.Lock()
	defer configMu.Unlock()
	b, err := os.ReadFile(configFilePath())
	if err != nil {
		return appConfig{}
	}
	var c appConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return appConfig{}
	}
	c.DestDir = strings.TrimSpace(c.DestDir)
	return c
}

func saveDestDir(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	configMu.Lock()
	defer configMu.Unlock()
	path := configFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	b, err := json.MarshalIndent(appConfig{DestDir: dir}, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, append(b, '\n'), 0o600)
}

func rememberedDownloadDir() string {
	c := loadConfig()
	if c.DestDir != "" {
		return c.DestDir
	}
	return defaultDownloadDir()
}
