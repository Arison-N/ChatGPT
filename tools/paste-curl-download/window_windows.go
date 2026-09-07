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

	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		appdata = os.TempDir()
	}
	dataDir := filepath.Join(appdata, "Zoom-loader", "webview2")
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
	uiDispatch = w.Dispatch
	defer func() { uiDispatch = nil }()
	defer w.Destroy()
	lockStandaloneWebView(w)
	w.Init(`document.addEventListener("keydown",function(e){if(e.altKey)return;var k=e.key,c=e.code,m=e.ctrlKey||e.metaKey;if(m&&(k==="+"||k==="-"||k==="="||k==="_"||k==="0"||c==="Equal"||c==="Minus"||c==="Digit0"||c==="NumpadAdd"||c==="NumpadSubtract"||c==="Numpad0")){e.preventDefault();return}if(m&&(k==="r"||k==="R"||k==="p"||k==="P"||k==="f"||k==="F"||k==="g"||k==="G"||k==="u"||k==="U"||k==="n"||k==="N"||k==="t"||k==="T"||k==="j"||k==="J")){e.preventDefault();return}if(k==="F3"||k==="F5"||k==="F7"||k==="F12")e.preventDefault()},true);document.addEventListener("wheel",function(e){if(e.ctrlKey)e.preventDefault()},{passive:false,capture:true});`)
	w.SetSize(920, 780, webview2.HintNone)
	w.Navigate(rawURL)
	w.Run()
}
