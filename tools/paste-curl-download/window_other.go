//go:build !windows

package main

func runUI(rawURL string, done <-chan struct{}) {
	runChromeWindow(rawURL, done)
}
