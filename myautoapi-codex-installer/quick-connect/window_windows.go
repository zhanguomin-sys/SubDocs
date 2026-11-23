package main

import (
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	assistantWindowClass = "MyAutoAPIQuickConnectWindow"
	assistantWindowWidth = 720
	startupWindowHeight  = 720
)

// Wails dimensions are logical units. Work in physical pixels here so the
// assistant keeps the same width even when Windows display scaling changes.
func fitAssistantWindow(contentHeight int, center bool) bool {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	user32 := windows.NewLazySystemDLL("user32.dll")
	setDPI := user32.NewProc("SetThreadDpiAwarenessContext")
	if setDPI.Find() != nil {
		return false
	}
	const perMonitorAwareV2 = ^uintptr(3) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
	previous, _, _ := setDPI.Call(perMonitorAwareV2)
	if previous == 0 {
		return false
	}
	defer setDPI.Call(previous)

	class, _ := windows.UTF16PtrFromString(assistantWindowClass)
	findWindow := user32.NewProc("FindWindowExW")
	windowPID := user32.NewProc("GetWindowThreadProcessId")
	var hwnd uintptr
	for {
		hwnd, _, _ = findWindow.Call(0, hwnd, uintptr(unsafe.Pointer(class)), 0)
		if hwnd == 0 {
			return false
		}
		var pid uint32
		windowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid == uint32(os.Getpid()) {
			break
		}
	}

	const nearestMonitor = 2
	monitor, _, _ := user32.NewProc("MonitorFromWindow").Call(hwnd, nearestMonitor)
	type rect struct{ left, top, right, bottom int32 }
	var info struct {
		size          uint32
		monitor, work rect
		flags         uint32
	}
	info.size = uint32(unsafe.Sizeof(info))
	ok, _, _ := user32.NewProc("GetMonitorInfoW").Call(monitor, uintptr(unsafe.Pointer(&info)))
	workWidth, workHeight := info.work.right-info.work.left, info.work.bottom-info.work.top
	if ok == 0 || workWidth <= 0 || workHeight <= 0 {
		return false
	}
	var frame, client rect
	getWindowRect := user32.NewProc("GetWindowRect")
	getClientRect := user32.NewProc("GetClientRect")
	frameOK, _, _ := getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&frame)))
	clientOK, _, _ := getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	if frameOK == 0 || clientOK == 0 {
		return false
	}
	currentWidth, currentHeight := frame.right-frame.left, frame.bottom-frame.top
	width, height := min(assistantWindowWidth, workWidth), min(startupWindowHeight, workHeight)
	if contentHeight > 0 {
		nonClient := max(0, currentHeight-(client.bottom-client.top))
		height = min(int32(contentHeight)+nonClient+2, workHeight)
	}
	x, y := frame.left, frame.top
	if center {
		x = info.work.left + (workWidth-width)/2
		y = info.work.top + (workHeight-height)/2
	}
	x = max(info.work.left, min(x, info.work.right-width))
	y = max(info.work.top, min(y, info.work.bottom-height))
	if width == currentWidth && height == currentHeight && x == frame.left && y == frame.top {
		return true
	}
	const noZOrder, noActivate = 0x0004, 0x0010
	ok, _, _ = user32.NewProc("SetWindowPos").Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(width), uintptr(height), noZOrder|noActivate)
	frameOK, _, _ = getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&frame)))
	return ok != 0 && frameOK != 0 && frame.right-frame.left == width && frame.bottom-frame.top == height
}
