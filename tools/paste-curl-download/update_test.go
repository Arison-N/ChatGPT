package main

import "testing"

func TestVersionNewer(t *testing.T) {
	if !versionNewer("1.2.1", "1.2.0") {
		t.Fatal("1.2.1 should be newer than 1.2.0")
	}
	if versionNewer("1.2.0", "1.2.0") {
		t.Fatal("same version is not newer")
	}
	if versionNewer("1.2.0", "1.10.0") {
		t.Fatal("1.2.0 should not be newer than 1.10.0")
	}
	if !versionNewer("2.0.0", "1.9.9") {
		t.Fatal("2.0.0 should be newer")
	}
}

func TestAllowedUpdateURL(t *testing.T) {
	ok := "https://raw.githubusercontent.com/Arison-N/ChatGPT/main/tools/paste-curl-download/dist/Zoom-loader.exe"
	if !allowedUpdateURL(ok) {
		t.Fatal("expected allow")
	}
	if allowedUpdateURL("https://evil.example/Zoom-loader.exe") {
		t.Fatal("expected deny")
	}
	if allowedUpdateURL("http://raw.githubusercontent.com/Arison-N/ChatGPT/main/x/Zoom-loader.exe") {
		t.Fatal("http should deny")
	}
}
