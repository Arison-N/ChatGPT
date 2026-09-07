package main

import "testing"

func TestWindowsParentDirUNC(t *testing.T) {
	in := `\\Arison_NAS\home\Arison\60_Learning\04_Bachelor-Degree\HKMU_Civil Engineering\2026-2027_Year 2\2026_Autumn Term\ENGG 2008SEF_Structural Analysis\10_Recordings\2026-09-07_ENGG 2008SEF_01.mp4`
	want := `\\Arison_NAS\home\Arison\60_Learning\04_Bachelor-Degree\HKMU_Civil Engineering\2026-2027_Year 2\2026_Autumn Term\ENGG 2008SEF_Structural Analysis\10_Recordings`
	if got := windowsParentDir(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := windowsParentDir(`\\Arison_NAS\home`); got != `\\Arison_NAS\home` {
		t.Fatalf("share root: %q", got)
	}
	if got := windowsParentDir(`C:\Users\kelvi\Downloads\a.mp4`); got != `C:\Users\kelvi\Downloads` {
		t.Fatalf("local: %q", got)
	}
}

func TestWindowsExtendedPathUNC(t *testing.T) {
	got := windowsExtendedPath(`\\Arison_NAS\home\Arison\file.mp4`)
	want := `\\?\UNC\Arison_NAS\home\Arison\file.mp4`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
