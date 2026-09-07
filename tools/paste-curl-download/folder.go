package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
)

func pickFolder(current string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return pickFolderWindows(current)
	case "darwin":
		return pickFolderDarwin(current)
	default:
		return pickFolderLinux(current)
	}
}

func pickFolderWindows(current string) (string, error) {
	dir, err := os.MkdirTemp("", "zoom-loader-pick-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	csPath := filepath.Join(dir, "folderpicker.cs")
	psPath := filepath.Join(dir, "pickfolder.ps1")
	outPath := filepath.Join(dir, "path.txt")
	if err := os.WriteFile(csPath, folderPickerCS, 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(psPath, pickFolderPS1, 0o644); err != nil {
		return "", err
	}

	cmd := exec.Command("powershell",
		"-NoProfile", "-STA", "-ExecutionPolicy", "Bypass",
		"-File", psPath,
		"-OutPath", outPath,
		"-Initial", current,
		"-CsPath", csPath,
	)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("folder picker: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		return "", nil // cancelled
	}
	path := strings.TrimSpace(string(raw))
	if path == "" {
		return "", nil
	}
	if !utf8.ValidString(path) {
		return "", fmt.Errorf("folder path is not valid UTF-8")
	}
	return path, nil
}

func pickFolderDarwin(current string) (string, error) {
	script := `POSIX path of (choose folder with prompt "File saving location")`
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func pickFolderLinux(current string) (string, error) {
	if p, err := exec.LookPath("zenity"); err == nil {
		cmd := exec.Command(p, "--file-selection", "--directory", "--title=File saving location", "--filename="+current+"/")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
	return "", fmt.Errorf("no folder picker available")
}

func revealInExplorer(path string) {
	if runtime.GOOS != "windows" {
		return
	}
	_ = exec.Command("explorer", "/select,"+filepath.Clean(path)).Start()
}
