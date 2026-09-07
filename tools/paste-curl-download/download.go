package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProgressFunc func(written, total int64)

func downloadParsed(parsed ParsedCurl, dest string, progress ProgressFunc) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, parsed.URL, nil)
	if err != nil {
		return err
	}
	for _, h := range parsed.Headers {
		name, value, ok := strings.Cut(h, ":")
		if !ok {
			continue
		}
		key := http.CanonicalHeaderKey(strings.TrimSpace(name))
		if strings.EqualFold(key, "Range") || strings.EqualFold(key, "Cookie") {
			continue
		}
		req.Header.Set(key, strings.TrimSpace(value))
	}
	if parsed.Cookie != "" {
		req.Header.Set("Cookie", parsed.Cookie)
	}
	req.Header.Set("Range", "bytes=0-")

	client := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 90 * time.Second,
			IdleConnTimeout:       120 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "https" {
				return fmt.Errorf("refusing non-https redirect")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		// bytes 0-123/456
		if i := strings.LastIndex(cr, "/"); i >= 0 {
			if n, err := strconv.ParseInt(strings.TrimSpace(cr[i+1:]), 10, 64); err == nil {
				total = n
			}
		}
	}

	part := dest + ".part"
	out, err := os.Create(part)
	if err != nil {
		return err
	}
	defer func() {
		out.Close()
		if err != nil {
			_ = os.Remove(part)
		}
	}()

	var written int64
	buf := make([]byte, 256*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				err = werr
				return err
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
			err = rerr
			return err
		}
	}
	if err = out.Close(); err != nil {
		return err
	}
	if err = os.Rename(part, dest); err != nil {
		return err
	}
	return nil
}
