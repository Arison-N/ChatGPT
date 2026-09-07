//go:build windows

package main

import (
	"log"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/jchv/go-webview2"
)

var (
	shell32                         = syscall.NewLazyDLL("shell32.dll")
	procSetCurrentProcessExplicitID = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
)

func setAppUserModelID() {
	id, err := syscall.UTF16PtrFromString("Arison.ZoomLoader")
	if err != nil {
		return
	}
	procSetCurrentProcessExplicitID.Call(uintptr(unsafe.Pointer(id)))
}

func runUI(rawURL string, done <-chan struct{}) {
	setAppUserModelID()

	dataDir := filepath.Join(os.TempDir(), "zoom-loader-webview2")
	_ = os.MkdirAll(dataDir, 0o700)

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  dataDir,
		WindowOptions: webview2.WindowOptions{
			Title:  "Zoom-loader",
			Width:  920,
			Height: 780,
			Center: true,
		},
	})
	if w == nil {
		log.Print("WebView2 unavailable; falling back to Chrome --app")
		runChromeWindow(rawURL, done)
		return
	}
	requestUIClose = func() { w.Terminate() }
	defer w.Destroy()
	w.SetSize(920, 780, webview2.HintNone)
	w.Navigate(rawURL)
	w.Run()
}
