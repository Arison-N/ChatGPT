package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func pickFolder(current, title string) (string, error) {
	if strings.TrimSpace(title) == "" {
		title = "File saving location"
	}
	switch runtime.GOOS {
	case "windows":
		return pickFolderWindows(current, title)
	case "darwin":
		return pickFolderDarwin(current, title)
	default:
		return pickFolderLinux(current, title)
	}
}

func pickFolderDarwin(current, title string) (string, error) {
	if title == "" {
		title = "File saving location"
	}
	script := `POSIX path of (choose folder with prompt "` + strings.ReplaceAll(title, `"`, `'`) + `")`
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func pickFolderLinux(current, title string) (string, error) {
	if title == "" {
		title = "File saving location"
	}
	if p, err := exec.LookPath("zenity"); err == nil {
		cmd := exec.Command(p, "--file-selection", "--directory", "--title="+title, "--filename="+current+"/")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
	return "", fmt.Errorf("no folder picker available")
}

func revealInExplorer(path string) {
	clean := filepath.Clean(path)
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("explorer", "/select,"+clean).Start()
	case "darwin":
		_ = exec.Command("open", "-R", clean).Start()
	default:
		_ = exec.Command("xdg-open", filepath.Dir(clean)).Start()
	}
}

func openFile(path string) {
	clean := filepath.Clean(path)
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("cmd", "/c", "start", "", clean).Start()
	case "darwin":
		_ = exec.Command("open", clean).Start()
	default:
		_ = exec.Command("xdg-open", clean).Start()
	}
}

func runAfterDownload(path, action string) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "open-file":
		openFile(path)
	case "open-location":
		revealInExplorer(path)
	}
}
