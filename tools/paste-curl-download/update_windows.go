//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func restartWithNewBinary(exePath, newPath string) error {
	ps1 := filepath.Join(os.TempDir(), "zoom-loader-apply.ps1")
	script := `
param($Exe, $New)
$ErrorActionPreference = 'Stop'
for ($i = 0; $i -lt 30; $i++) {
  Start-Sleep -Milliseconds 400
  try {
    Move-Item -LiteralPath $New -Destination $Exe -Force
    break
  } catch {}
}
Start-Process -FilePath $Exe
Remove-Item -LiteralPath $MyInvocation.MyCommand.Path -Force -ErrorAction SilentlyContinue
`
	if err := os.WriteFile(ps1, []byte(script), 0o644); err != nil {
		return err
	}
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", ps1, "-Exe", exePath, "-New", newPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
