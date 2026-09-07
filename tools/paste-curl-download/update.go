package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type remoteVersion struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	Exe     string `json:"exe"`
}

func parseVer(s string) []int {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	parts := strings.Split(s, ".")
	out := make([]int, 0, 3)
	for i := 0; i < 3; i++ {
		n := 0
		if i < len(parts) {
			n, _ = strconv.Atoi(parts[i])
		}
		out = append(out, n)
	}
	return out
}

func versionNewer(latest, current string) bool {
	a, b := parseVer(latest), parseVer(current)
	for i := 0; i < 3; i++ {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
	}
	return false
}

func allowedUpdateURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Host)
	if host != "raw.githubusercontent.com" && host != "github.com" {
		return false
	}
	p := u.Path
	return strings.Contains(p, "Arison-N/ChatGPT") && strings.HasSuffix(p, "Zoom-loader.exe")
}

func httpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 20 * time.Second,
		},
	}
}

func fetchRemoteVersion() (remoteVersion, string, error) {
	client := httpClient(25 * time.Second)
	var last error
	for _, raw := range versionManifestURLs {
		req, err := http.NewRequest(http.MethodGet, raw, nil)
		if err != nil {
			last = err
			continue
		}
		req.Header.Set("User-Agent", "Zoom-loader/"+Version)
		req.Header.Set("Cache-Control", "no-cache")
		resp, err := client.Do(req)
		if err != nil {
			last = err
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			last = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			last = fmt.Errorf("%s: HTTP %d", raw, resp.StatusCode)
			continue
		}
		var v remoteVersion
		if err := json.Unmarshal(body, &v); err != nil {
			last = err
			continue
		}
		if strings.TrimSpace(v.Version) == "" || strings.TrimSpace(v.Exe) == "" {
			last = fmt.Errorf("invalid version manifest")
			continue
		}
		if !allowedUpdateURL(v.Exe) {
			last = fmt.Errorf("update URL is not allowed")
			continue
		}
		return v, raw, nil
	}
	if last == nil {
		last = fmt.Errorf("no update manifest found")
	}
	return remoteVersion{}, "", last
}

func downloadUpdate(exeURL, dest string, progress func(written, total int64)) error {
	if !allowedUpdateURL(exeURL) {
		return fmt.Errorf("update URL is not allowed")
	}
	client := httpClient(5 * time.Minute)
	req, err := http.NewRequest(http.MethodGet, exeURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Zoom-loader/"+Version)
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	total := resp.ContentLength
	buf := make([]byte, 64*1024)
	var written int64
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if progress != nil {
				progress(written, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	return out.Close()
}
