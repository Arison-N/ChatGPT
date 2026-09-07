package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
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

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func pickFolderWindows(current string) (string, error) {
	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$d = New-Object System.Windows.Forms.FolderBrowserDialog
$d.Description = 'File saving location'
$d.ShowNewFolderButton = $true
$d.SelectedPath = %s
$r = $d.ShowDialog()
if ($r -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::Out.Write($d.SelectedPath) }
`, psQuote(current))
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-Command", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("folder picker: %v %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
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
