//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

var (
	ole32    = syscall.NewLazyDLL("ole32.dll")
	shell32p = syscall.NewLazyDLL("shell32.dll")

	procCoInitializeEx              = ole32.NewProc("CoInitializeEx")
	procCoUninitialize              = ole32.NewProc("CoUninitialize")
	procCoCreateInstance            = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree               = ole32.NewProc("CoTaskMemFree")
	procSHCreateItemFromParsingName = shell32p.NewProc("SHCreateItemFromParsingName")
	procSHParseDisplayName          = shell32p.NewProc("SHParseDisplayName")
	procILCreateFromPathW           = shell32p.NewProc("ILCreateFromPathW")
	procILClone                     = shell32p.NewProc("ILClone")
	procILRemoveLastID              = shell32p.NewProc("ILRemoveLastID")
	procILFindLastID                = shell32p.NewProc("ILFindLastID")
	procSHOpenFolderAndSelectItems  = shell32p.NewProc("SHOpenFolderAndSelectItems")
	procILFree                      = shell32p.NewProc("ILFree")
	procShellExecuteW               = shell32p.NewProc("ShellExecuteW")
	user32                          = syscall.NewLazyDLL("user32.dll")
	procAllowSetForegroundWindow    = user32.NewProc("AllowSetForegroundWindow")
)

const (
	coInitApartmentThreaded = 0x2
	clsctxInprocServer      = 0x1
	fosPickFolders          = 0x00000020
	fosForceFileSystem      = 0x00000040
	fosPathMustExist        = 0x00000800
	sigdnFileSysPath        = 0x80058000
	rpcEChangedMode         = 0x80010106
	hresultCancelled        = 0x800704C7
	swShowNormal            = 1
	asfwAny                 = ^uintptr(0)
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	clsidFileOpenDialog = guid{0xDC1C5A9C, 0xE88A, 0x4DDE, [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileOpenDialog  = guid{0xD57C7288, 0xD4AD, 0x4768, [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidIShellItem       = guid{0x43826D1E, 0xE718, 0x42EE, [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}
)

type iUnknown struct{ vt *iFileOpenDialogVtbl }

type iFileOpenDialogVtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	Show                uintptr
	SetFileTypes        uintptr
	SetFileTypeIndex    uintptr
	GetFileTypeIndex    uintptr
	Advise              uintptr
	Unadvise            uintptr
	SetOptions          uintptr
	GetOptions          uintptr
	SetDefaultFolder    uintptr
	SetFolder           uintptr
	GetFolder           uintptr
	GetCurrentSelection uintptr
	SetFileName         uintptr
	GetFileName         uintptr
	SetTitle            uintptr
	SetOkButtonLabel    uintptr
	SetFileNameLabel    uintptr
	GetResult           uintptr
	AddPlace            uintptr
	SetDefaultExtension uintptr
	Close               uintptr
	SetClientGuid       uintptr
	ClearClientData     uintptr
	SetFilter           uintptr
	GetResults          uintptr
	GetSelectedItems    uintptr
}

type iShellItem struct{ vt *iShellItemVtbl }

type iShellItemVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	BindToHandler  uintptr
	GetParent      uintptr
	GetDisplayName uintptr
	GetAttributes  uintptr
	Compare        uintptr
}

func hrFail(r uintptr) bool { return int32(r) < 0 }

func releaseCOM(vtRelease uintptr, obj unsafe.Pointer) {
	if obj != nil && vtRelease != 0 {
		syscall.SyscallN(vtRelease, uintptr(obj))
	}
}

func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	n := 0
	for {
		c := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n)*2))
		if c == 0 {
			break
		}
		n++
	}
	s := unsafe.Slice(p, n)
	return string(utf16.Decode(s))
}

func pickFolderCOM(current, title string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr, _, _ := procCoInitializeEx.Call(0, coInitApartmentThreaded)
	if hrFail(hr) && uint32(hr) != rpcEChangedMode {
		return "", fmt.Errorf("CoInitializeEx: 0x%X", uint32(hr))
	}
	uninit := !hrFail(hr)
	if uninit {
		defer procCoUninitialize.Call()
	}

	var dlg *iUnknown
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIFileOpenDialog)),
		uintptr(unsafe.Pointer(&dlg)),
	)
	if hrFail(hr) || dlg == nil {
		return "", fmt.Errorf("CoCreateInstance: 0x%X", hr)
	}
	defer releaseCOM(dlg.vt.Release, unsafe.Pointer(dlg))

	var opts uint32
	if hr, _, _ = syscall.SyscallN(dlg.vt.GetOptions, uintptr(unsafe.Pointer(dlg)), uintptr(unsafe.Pointer(&opts))); hrFail(hr) {
		return "", fmt.Errorf("GetOptions: 0x%X", hr)
	}
	opts |= fosPickFolders | fosForceFileSystem | fosPathMustExist
	if hr, _, _ = syscall.SyscallN(dlg.vt.SetOptions, uintptr(unsafe.Pointer(dlg)), uintptr(opts)); hrFail(hr) {
		return "", fmt.Errorf("SetOptions: 0x%X", hr)
	}

	if title != "" {
		w, err := syscall.UTF16PtrFromString(title)
		if err == nil {
			syscall.SyscallN(dlg.vt.SetTitle, uintptr(unsafe.Pointer(dlg)), uintptr(unsafe.Pointer(w)))
		}
	}
	if current != "" {
		w, err := syscall.UTF16PtrFromString(current)
		if err == nil {
			var folder *iShellItem
			iid := iidIShellItem
			hr, _, _ = procSHCreateItemFromParsingName.Call(
				uintptr(unsafe.Pointer(w)),
				0,
				uintptr(unsafe.Pointer(&iid)),
				uintptr(unsafe.Pointer(&folder)),
			)
			if !hrFail(hr) && folder != nil {
				syscall.SyscallN(dlg.vt.SetFolder, uintptr(unsafe.Pointer(dlg)), uintptr(unsafe.Pointer(folder)))
				releaseCOM(folder.vt.Release, unsafe.Pointer(folder))
			}
		}
	}

	hr, _, _ = syscall.SyscallN(dlg.vt.Show, uintptr(unsafe.Pointer(dlg)), 0)
	if uint32(hr) == hresultCancelled {
		return "", nil
	}
	if hrFail(hr) {
		return "", fmt.Errorf("Show: 0x%X", uint32(hr))
	}

	var item *iShellItem
	if hr, _, _ = syscall.SyscallN(dlg.vt.GetResult, uintptr(unsafe.Pointer(dlg)), uintptr(unsafe.Pointer(&item))); hrFail(hr) || item == nil {
		return "", nil
	}
	defer releaseCOM(item.vt.Release, unsafe.Pointer(item))

	var namePtr *uint16
	if hr, _, _ = syscall.SyscallN(item.vt.GetDisplayName, uintptr(unsafe.Pointer(item)), uintptr(sigdnFileSysPath), uintptr(unsafe.Pointer(&namePtr))); hrFail(hr) || namePtr == nil {
		return "", fmt.Errorf("GetDisplayName: 0x%X", hr)
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(namePtr)))

	path := utf16PtrToString(namePtr)
	if path == "" || !utf8.ValidString(path) {
		return "", fmt.Errorf("folder path is not valid UTF-8")
	}
	return path, nil
}

func pickFolderWindows(current, title string) (string, error) {
	path, err := pickFolderCOM(current, title)
	if err == nil {
		return path, nil
	}
	return pickFolderWindowsPowerShell(current, title)
}

func pickFolderWindowsPowerShell(current, title string) (string, error) {
	dir, err := os.MkdirTemp("", "zoom-loader-pick-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	csPath := filepath.Join(dir, "folderpicker.cs")
	psPath := filepath.Join(dir, "pickfolder.ps1")
	outPath := filepath.Join(dir, "path.txt")
	if err := os.WriteFile(csPath, folderPickerCS, 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(psPath, pickFolderPS1, 0o644); err != nil {
		return "", err
	}

	cmd := exec.Command("powershell.exe",
		"-NoLogo", "-NonInteractive", "-NoProfile", "-STA",
		"-WindowStyle", "Hidden",
		"-ExecutionPolicy", "Bypass",
		"-File", psPath,
		"-OutPath", outPath,
		"-Initial", current,
		"-CsPath", csPath,
		"-Title", title,
	)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("folder picker: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		return "", nil
	}
	path := strings.TrimSpace(string(raw))
	if path == "" {
		return "", nil
	}
	if !utf8.ValidString(path) {
		return "", fmt.Errorf("folder path is not valid UTF-8")
	}
	return path, nil
}

func parsePIDL(path string) uintptr {
	w, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	var pidl uintptr
	hr, _, _ := procSHParseDisplayName.Call(
		uintptr(unsafe.Pointer(w)),
		0,
		uintptr(unsafe.Pointer(&pidl)),
		0,
		0,
	)
	if !hrFail(hr) && pidl != 0 {
		return pidl
	}
	pidl, _, _ = procILCreateFromPathW.Call(uintptr(unsafe.Pointer(w)))
	return pidl
}

func openAndSelectPIDL(pidl uintptr) bool {
	if pidl == 0 {
		return false
	}
	parent, _, _ := procILClone.Call(pidl)
	if parent == 0 {
		hr, _, _ := procSHOpenFolderAndSelectItems.Call(pidl, 0, 0, 0)
		return !hrFail(hr)
	}
	defer procILFree.Call(parent)
	ok, _, _ := procILRemoveLastID.Call(parent)
	if ok == 0 {
		hr, _, _ := procSHOpenFolderAndSelectItems.Call(pidl, 0, 0, 0)
		return !hrFail(hr)
	}
	child, _, _ := procILFindLastID.Call(pidl)
	if child == 0 {
		return false
	}
	apidl := child
	hr, _, _ := procSHOpenFolderAndSelectItems.Call(parent, 1, uintptr(unsafe.Pointer(&apidl)), 0)
	return !hrFail(hr)
}

func shellOpen(path string) bool {
	if path == "" {
		return false
	}
	w, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	verb, _ := syscall.UTF16PtrFromString("open")
	r, _, _ := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(w)), 0, 0, swShowNormal)
	return r > 32
}

func revealInExplorerWindows(path string) {
	path = windowsNormPath(path)
	if path == "" {
		return
	}
	go func() {
		time.Sleep(250 * time.Millisecond)
		runOnUI(func() { revealInExplorerWindowsSync(path) })
	}()
}

func revealInExplorerWindowsSync(path string) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coInitApartmentThreaded)
	if !hrFail(hr) {
		defer procCoUninitialize.Call()
	}
	procAllowSetForegroundWindow.Call(asfwAny)

	candidates := []string{path, windowsExtendedPath(path)}
	for i := 0; i < 6; i++ {
		for _, p := range candidates {
			pidl := parsePIDL(p)
			if pidl == 0 {
				continue
			}
			ok := openAndSelectPIDL(pidl)
			procILFree.Call(pidl)
			if ok {
				return
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = shellOpen(windowsParentDir(path))
}

func openFileWindows(path string) {
	path = windowsNormPath(path)
	if path == "" {
		return
	}
	go func() {
		time.Sleep(250 * time.Millisecond)
		runOnUI(func() {
			procAllowSetForegroundWindow.Call(asfwAny)
			if shellOpen(path) {
				return
			}
			_ = shellOpen(windowsExtendedPath(path))
		})
	}()
}
