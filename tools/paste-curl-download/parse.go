package main

import (
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"
	"unicode"
)

var (
	headerSingle  = regexp.MustCompile(`(?is)(?:-H|--header)\s+'([^']*)'`)
	headerDouble  = regexp.MustCompile(`(?is)(?:-H|--header)\s+"([^"]*)"`)
	cookieSingle  = regexp.MustCompile(`(?is)(?:-b|--cookie)\s+'([^']*)'`)
	cookieDouble  = regexp.MustCompile(`(?is)(?:-b|--cookie)\s+"([^"]*)"`)
	urlFlagSingle = regexp.MustCompile(`(?is)--url\s+'(https://[^']+)'`)
	urlFlagDouble = regexp.MustCompile(`(?is)--url\s+"(https://[^"]+)"`)
	urlCurlSingle = regexp.MustCompile(`(?i)\bcurl(?:\.exe)?\s+(?:-[A-Za-z]\s+)*'(https://[^']+)'`)
	urlCurlDouble = regexp.MustCompile(`(?i)\bcurl(?:\.exe)?\s+(?:-[A-Za-z]\s+)*"(https://[^"]+)"`)
	bareHTTPSRe   = regexp.MustCompile(`https://[^\s'"]+`)
	promptRe      = regexp.MustCompile(`(?m)^\$\s*`)
	cdAndRe       = regexp.MustCompile(`^(?:cd\s+\S+\s*&&\s*)+`)
	dirAndRe      = regexp.MustCompile(`^(?:~/[^\s]*|/[^\s]+|[A-Za-z]:\\[^\s]+)\s*&&\s*`)
	bashContRe    = regexp.MustCompile(`\\\s*\n`)
	cmdContRe     = regexp.MustCompile(`\^\s*\n`)
	illegalName   = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
)

type ParsedCurl struct {
	URL      string
	Headers  []string
	Cookie   string
	Filename string
}

func normalizePaste(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text)
	text = promptRe.ReplaceAllString(text, "")
	text = cdAndRe.ReplaceAllString(text, "")
	text = dirAndRe.ReplaceAllString(text, "")
	text = bashContRe.ReplaceAllString(text, " ")
	text = cmdContRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func filenameFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "download.mp4"
	}
	name := path.Base(u.Path)
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "download.mp4"
	}
	name = illegalName.ReplaceAllString(name, "_")
	return name
}

// LectureFilename turns a user-typed lecture title into a safe Windows filename.
// Empty input falls back to detectedName. Missing extension becomes .mp4.
func LectureFilename(userName, detectedName string) (string, error) {
	name := strings.TrimSpace(userName)
	if name == "" {
		name = strings.TrimSpace(detectedName)
	}
	if name == "" {
		return "", errors.New("請輸入課堂檔名")
	}
	name = strings.ReplaceAll(name, "\\", "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	name = strings.TrimSpace(name)
	name = illegalName.ReplaceAllString(name, "_")
	name = strings.TrimRightFunc(name, unicode.IsSpace)
	if name == "" || name == "." || name == ".." {
		return "", errors.New("課堂檔名無效")
	}
	if path.Ext(name) == "" {
		name += ".mp4"
	}
	return name, nil
}

func dedupeHeaders(headers []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(headers))
	for _, raw := range headers {
		raw = strings.TrimSpace(raw)
		key := raw
		if i := strings.Index(raw, ":"); i >= 0 {
			key = strings.ToLower(strings.TrimSpace(raw[:i]))
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, raw)
	}
	return out
}

func ParseCurlPaste(text string) (ParsedCurl, error) {
	blob := normalizePaste(text)
	if blob == "" {
		return ParsedCurl{}, errors.New("Empty paste.")
	}

	urlStr := firstGroup(urlFlagSingle, blob)
	if urlStr == "" {
		urlStr = firstGroup(urlFlagDouble, blob)
	}
	if urlStr == "" {
		urlStr = firstGroup(urlCurlSingle, blob)
	}
	if urlStr == "" {
		urlStr = firstGroup(urlCurlDouble, blob)
	}
	if urlStr == "" {
		found := bareHTTPSRe.FindAllString(blob, -1)
		for _, u := range found {
			if len(u) > len(urlStr) {
				urlStr = u
			}
		}
	}
	urlStr = strings.TrimSpace(strings.TrimRight(urlStr, "\\"))
	if !strings.HasPrefix(urlStr, "https://") {
		return ParsedCurl{}, errors.New("No https URL found. Paste Chrome Copy as cURL (bash).")
	}

	headers := append(allGroups(headerSingle, blob), allGroups(headerDouble, blob)...)
	cookies := append(allGroups(cookieSingle, blob), allGroups(cookieDouble, blob)...)
	cookie := ""
	if len(cookies) > 0 {
		cookie = cookies[len(cookies)-1]
	}

	rewritten := make([]string, 0, len(headers)+1)
	hasRange := false
	for _, h := range headers {
		name, value, ok := strings.Cut(h, ":")
		lname := strings.ToLower(strings.TrimSpace(name))
		if lname == "range" {
			rewritten = append(rewritten, "Range: bytes=0-")
			hasRange = true
			continue
		}
		if lname == "cookie" && cookie == "" && ok {
			cookie = strings.TrimSpace(value)
		}
		rewritten = append(rewritten, h)
	}
	if !hasRange {
		rewritten = append(rewritten, "Range: bytes=0-")
	}

	return ParsedCurl{
		URL:      urlStr,
		Headers:  dedupeHeaders(rewritten),
		Cookie:   cookie,
		Filename: filenameFromURL(urlStr),
	}, nil
}

func firstGroup(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func allGroups(re *regexp.Regexp, s string) []string {
	ms := re.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		if len(m) >= 2 {
			out = append(out, strings.TrimSpace(m[1]))
		}
	}
	return out
}
