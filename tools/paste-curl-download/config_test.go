package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveDestDirChinese(t *testing.T) {
	dir := t.TempDir()
	configPathForTest = filepath.Join(dir, "config.json")
	t.Cleanup(func() { configPathForTest = "" })

	want := `C:\Users\kelvi\OneDrive\文件\課堂錄影`
	saveDestDir(want)
	got := rememberedDownloadDir()
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	raw, err := os.ReadFile(configPathForTest)
	if err != nil {
		t.Fatal(err)
	}
	if !stringsContainsRune(string(raw), '課') {
		t.Fatalf("config file lost Chinese text: %s", raw)
	}
}

func stringsContainsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
