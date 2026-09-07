package main

import (
	"strings"
	"testing"
)

const bash = `
curl --url 'https://ssrweb.zoom.us/replay02/2026/09/04/GMT20260904-005633_Recording_1920x1080.mp4?data=abc&tid=v=2.0;clid=aw1' \
  -H 'Accept: */*' \
  -H 'Range: bytes=0-4128767' \
  -H 'Referer: https://hkmu.zoom.us/' \
  -H 'sec-ch-ua: "Chromium";v="152", "Not?A_Brand";v="24"' \
  -b '_zm_ssid=aw1_c_test; cf_clearance=abc|1|0|def'
`

const cmd = `
curl "https://example.com/files/lecture.mp4?x=1" ^
  -H "Range: bytes=100-200" ^
  -H "Referer: https://hkmu.zoom.us/" ^
  -b "sid=1"
`

const prefixed = `
~/Downloads && curl -L --fail -o GMT20260904-005633_Recording_1920x1080.mp4 --url 'https://ssrweb.zoom.us/a.mp4?q=1' \
  -H 'Range: bytes=0-99'
`

const chromeReal = `
curl --url 'https://ssrweb.zoom.us/replay02/2026/09/04/1203006F-1514-4E01-951A-BE706105DB4F/GMT20260904-005633_Recording_1920x1080.mp4?response-content-type=video%2Fmp4&tid=v=2.0;clid=aw1&Key-Pair-Id=K1YYCGW8V4AHXW' \
  -H 'Accept: */*' \
  -H 'Connection: keep-alive' \
  -b '_zm_ssid=aw1_c_test; cf_clearance=abc|1|0|def' \
  -H 'Range: bytes=0-' \
  -H 'Referer: https://hkmu.zoom.us/' \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36' \
  -H 'sec-ch-ua: "Chromium";v="152", "Not?A_Brand";v="24", "Google Chrome";v="152"' \
  -H 'sec-ch-ua-mobile: ?0' \
  -H 'sec-ch-ua-platform: "Windows"'
`

func TestParseBashZoom(t *testing.T) {
	p, err := ParseCurlPaste(bash)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p.URL, "https://ssrweb.zoom.us/") {
		t.Fatalf("url %q", p.URL)
	}
	if p.Filename != "GMT20260904-005633_Recording_1920x1080.mp4" {
		t.Fatalf("filename %q", p.Filename)
	}
	if p.Cookie != "_zm_ssid=aw1_c_test; cf_clearance=abc|1|0|def" {
		t.Fatalf("cookie %q", p.Cookie)
	}
	if !headerEq(p.Headers, "Range: bytes=0-") {
		t.Fatalf("headers %#v", p.Headers)
	}
}

func TestParseChromeReal(t *testing.T) {
	p, err := ParseCurlPaste(chromeReal)
	if err != nil {
		t.Fatal(err)
	}
	if p.Filename != "GMT20260904-005633_Recording_1920x1080.mp4" {
		t.Fatalf("filename %q", p.Filename)
	}
	if !strings.Contains(p.URL, "Key-Pair-Id=K1YYCGW8V4AHXW") || !strings.Contains(p.URL, "tid=v=2.0;clid=aw1") {
		t.Fatalf("url %q", p.URL)
	}
}

func TestParseCmd(t *testing.T) {
	p, err := ParseCurlPaste(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if p.Filename != "lecture.mp4" {
		t.Fatalf("filename %q", p.Filename)
	}
	if p.Cookie != "sid=1" {
		t.Fatalf("cookie %q", p.Cookie)
	}
}

func TestParsePrefixed(t *testing.T) {
	p, err := ParseCurlPaste(prefixed)
	if err != nil {
		t.Fatal(err)
	}
	if p.Filename != "a.mp4" {
		t.Fatalf("filename %q", p.Filename)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := ParseCurlPaste("   "); err == nil {
		t.Fatal("expected empty error")
	}
	if _, err := ParseCurlPaste("curl http://evil.example/x.mp4"); err == nil {
		t.Fatal("expected https error")
	}
}

func TestLectureFilename(t *testing.T) {
	got, err := LectureFilename("2026-09-04 微積分 L1", "GMT.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-09-04 微積分 L1.mp4" {
		t.Fatalf("got %q", got)
	}
	got, err = LectureFilename("notes.mkv", "GMT.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if got != "notes.mkv" {
		t.Fatalf("got %q", got)
	}
	got, err = LectureFilename("", "GMT20260904.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if got != "GMT20260904.mp4" {
		t.Fatalf("got %q", got)
	}
	got, err = LectureFilename(`C:\evil\a<>b`, "x.mp4")
	if err != nil {
		t.Fatal(err)
	}
	if got != "a__b.mp4" {
		t.Fatalf("got %q", got)
	}
	if _, err := LectureFilename("", ""); err == nil {
		t.Fatal("expected empty name error")
	}
}

func headerEq(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
