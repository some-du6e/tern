package main

import (
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
	"golang.org/x/sys/windows"
)

// Windows keeps its own title bar, dark or light with the theme.
const (
	controlsLeft = 12
	frameless    = false
)

var windowOptions []app.Option

var (
	dwmSetWindowAttribute = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
	hwnd                  uintptr
	wantDark              bool
)

// dwmwaUseImmersiveDarkMode is DWMWA_USE_IMMERSIVE_DARK_MODE (Windows 10
// 20H1 and later).
const dwmwaUseImmersiveDarkMode = 20

func platformEvent(e event.Event) {
	if v, ok := e.(app.Win32ViewEvent); ok && v.HWND != 0 {
		hwnd = v.HWND
		titleDark(wantDark)
	}
}

// titleDark paints the native title bar dark or light.
func titleDark(dark bool) {
	wantDark = dark
	if hwnd == 0 || dwmSetWindowAttribute.Find() != nil {
		return
	}
	on := int32(0)
	if dark {
		on = 1
	}
	dwmSetWindowAttribute.Call(hwnd, dwmwaUseImmersiveDarkMode, uintptr(unsafe.Pointer(&on)), unsafe.Sizeof(on))
}
