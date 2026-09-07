//go:build !windows

package main

import (
	"os"
	"os/exec"
)

func restartWithNewBinary(exePath, newPath string) error {
	if err := os.Rename(newPath, exePath); err != nil {
		return err
	}
	return exec.Command(exePath).Start()
}
