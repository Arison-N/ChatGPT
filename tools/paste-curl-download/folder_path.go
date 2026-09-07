package main

import "strings"

func windowsNormPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "/", "\\")
	return p
}

func windowsParentDir(p string) string {
	p = windowsNormPath(p)
	p = strings.TrimRight(p, `\`)
	i := strings.LastIndex(p, `\`)
	if i <= 1 {
		return p
	}
	if strings.HasPrefix(p, `\\`) {
		rest := strings.TrimPrefix(p, `\\`)
		if strings.Count(rest, `\`) <= 1 {
			return p
		}
	}
	return p[:i]
}

func windowsExtendedPath(p string) string {
	p = windowsNormPath(p)
	if p == "" || strings.HasPrefix(p, `\\?\`) {
		return p
	}
	if strings.HasPrefix(p, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(p, `\\`)
	}
	if len(p) >= 2 && p[1] == ':' {
		return `\\?\` + p
	}
	return p
}

var uiDispatch func(func())

func runOnUI(f func()) {
	if d := uiDispatch; d != nil {
		d(f)
		return
	}
	f()
}
