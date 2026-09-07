//go:build !windows

package main

import "errors"

func pickFolderWindows(current, title string) (string, error) {
	return "", errors.New("windows folder picker is only available on Windows")
}
