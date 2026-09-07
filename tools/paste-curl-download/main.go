package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"embed"
)

//go:embed web/index.html
var webFS embed.FS

type parseReq struct {
	Paste string `json:"paste"`
}

type downloadReq struct {
	Paste    string `json:"paste"`
	Filename string `json:"filename"`
	DestDir  string `json:"destDir"`
}

func defaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	d := filepath.Join(home, "Downloads")
	if st, err := os.Stat(d); err == nil && st.IsDir() {
		return d
	}
	return home
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func findChrome() string {
	var cands []string
	switch runtime.GOOS {
	case "windows":
		local := os.Getenv("LOCALAPPDATA")
		pf := os.Getenv("PROGRAMFILES")
		pf86 := os.Getenv("PROGRAMFILES(X86)")
		cands = []string{
			filepath.Join(local, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(pf, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(pf86, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(local, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`),
		}
	case "darwin":
		cands = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default:
		cands = []string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge"}
	}
	for _, c := range cands {
		if runtime.GOOS == "windows" || strings.Contains(c, "/") {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return c
			}
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}

func openAppWindow(rawURL string) (cmd *exec.Cmd, waitForExit bool, err error) {
	chrome := findChrome()
	if chrome != "" {
		dir, err := os.MkdirTemp("", "paste-curl-chrome-*")
		if err != nil {
			return nil, false, err
		}
		cmd := exec.Command(chrome,
			"--app="+rawURL,
			"--user-data-dir="+dir,
			"--window-size=920,780",
			"--no-first-run",
			"--no-default-browser-check",
		)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return nil, false, err
		}
		return cmd, true, nil
	}
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "start", "", rawURL)
		return cmd, false, cmd.Start()
	case "darwin":
		cmd := exec.Command("open", rawURL)
		return cmd, false, cmd.Start()
	default:
		cmd := exec.Command("xdg-open", rawURL)
		return cmd, false, cmd.Start()
	}
}

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal("listen: " + err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var once sync.Once
	shutdown := func() { once.Do(cancel) }

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, err := fs.ReadFile(webFS, "web/index.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/defaults", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"downloadDir": defaultDownloadDir()})
	})
	mux.HandleFunc("/api/pick-folder", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Current string `json:"current"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		path, err := pickFolder(strings.TrimSpace(req.Current))
		if err != nil {
			writeJSON(w, 200, map[string]any{"cancelled": true, "error": err.Error()})
			return
		}
		if path == "" {
			writeJSON(w, 200, map[string]any{"cancelled": true})
			return
		}
		writeJSON(w, 200, map[string]string{"path": path})
	})
	mux.HandleFunc("/api/parse", func(w http.ResponseWriter, r *http.Request) {
		var req parseReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		p, err := ParseCurlPaste(req.Paste)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]string{
			"filename": p.Filename,
			"url":      strings.SplitN(p.URL, "?", 2)[0],
		})
	})
	mux.HandleFunc("/api/download", func(w http.ResponseWriter, r *http.Request) {
		var req downloadReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		p, err := ParseCurlPaste(req.Paste)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		name, err := LectureFilename(req.Filename, p.Filename)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		dir := strings.TrimSpace(req.DestDir)
		if dir == "" {
			dir = defaultDownloadDir()
		}
		dest := filepath.Join(dir, name)

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Cache-Control", "no-cache")
		fl, _ := w.(http.Flusher)
		enc := json.NewEncoder(w)
		send := func(v any) {
			_ = enc.Encode(v)
			if fl != nil {
				fl.Flush()
			}
		}

		last := time.Now()
		err = downloadParsed(p, dest, func(written, total int64) {
			if time.Since(last) < 200*time.Millisecond && written != total {
				return
			}
			last = time.Now()
			send(map[string]any{"type": "progress", "written": written, "total": total})
		})
		if err != nil {
			send(map[string]any{"type": "error", "message": err.Error()})
			return
		}
		st, err := os.Stat(dest)
		if err != nil {
			send(map[string]any{"type": "error", "message": err.Error()})
			return
		}
		send(map[string]any{"type": "done", "path": dest, "bytes": st.Size()})
	})
	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
		go func() {
			time.Sleep(300 * time.Millisecond)
			shutdown()
		}()
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Print(err)
			shutdown()
		}
	}()

	rawURL := "http://" + ln.Addr().String() + "/"
	cmd, waitForExit, err := openAppWindow(rawURL)
	if err != nil {
		fatal("open window: " + err.Error())
	}

	if waitForExit && cmd != nil && cmd.Process != nil {
		go func() {
			_ = cmd.Wait()
			shutdown()
		}()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-ctx.Done():
	case <-sig:
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		shutdown()
	}
	_ = srv.Shutdown(context.Background())
}

func fatal(msg string) {
	notify(msg)
	os.Exit(1)
}
