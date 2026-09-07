//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

func notify(msg string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	text, _ := syscall.UTF16PtrFromString(msg)
	title, _ := syscall.UTF16PtrFromString("Zoom-loader")
	proc.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x00000010)
}
