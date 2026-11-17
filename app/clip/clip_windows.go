package clip

import (
	"bytes"
	"errors"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32  = windows.NewLazySystemDLL("user32.dll")
	kernel  = windows.NewLazySystemDLL("kernel32.dll")
	shell32 = windows.NewLazySystemDLL("shell32.dll")

	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	formatAvailable  = user32.NewProc("IsClipboardFormatAvailable")
	getClipboardData = user32.NewProc("GetClipboardData")
	registerFormat   = user32.NewProc("RegisterClipboardFormatW")
	globalLock       = kernel.NewProc("GlobalLock")
	globalUnlock     = kernel.NewProc("GlobalUnlock")
	globalSize       = kernel.NewProc("GlobalSize")
	dragQueryFile    = shell32.NewProc("DragQueryFileW")
)

const (
	cfText  = 13 // CF_UNICODETEXT
	cfDIB   = 8
	cfHDROP = 15
	cfDIBV5 = 17
)

func read() ([]Image, error) {
	// Another program can hold the clipboard for a moment.
	opened := false
	for i := 0; i < 5 && !opened; i++ {
		if r, _, _ := openClipboard.Call(0); r != 0 {
			opened = true
		} else {
			time.Sleep(15 * time.Millisecond)
		}
	}
	if !opened {
		return nil, errors.New("the clipboard is busy")
	}
	defer closeClipboard.Call()

	// Browsers and the Snipping Tool put a "PNG" format up, with alpha.
	name, _ := windows.UTF16PtrFromString("PNG")
	if f, _, _ := registerFormat.Call(uintptr(unsafe.Pointer(name))); f != 0 && available(f) {
		if b := global(f); len(b) > 0 {
			if imgs, err := fromPNG(b); err != ErrNone {
				return imgs, err
			}
		}
	}
	if available(cfHDROP) {
		if h, _, _ := getClipboardData.Call(cfHDROP); h != 0 {
			if imgs, err := fromFiles(dropFiles(h)); err != ErrNone {
				return imgs, err
			}
		}
	}
	// Office copies a bitmap of the cells or words along with their text.
	// The text is what was meant.
	if available(cfText) {
		return nil, ErrNone
	}
	for _, f := range []uintptr{cfDIBV5, cfDIB} {
		if !available(f) {
			continue
		}
		b := global(f)
		if len(b) == 0 {
			continue
		}
		img, err := DIB(b)
		if err != nil {
			continue
		}
		im, err := Encode(img, "pasted")
		return []Image{im}, err
	}
	return nil, ErrNone
}

func available(f uintptr) bool {
	r, _, _ := formatAvailable.Call(f)
	return r != 0
}

// global copies the clipboard's memory block for format f.
func global(f uintptr) []byte {
	h, _, _ := getClipboardData.Call(f)
	if h == 0 {
		return nil
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		return nil
	}
	defer globalUnlock.Call(h)
	n, _, _ := globalSize.Call(h)
	if n == 0 || n > 64<<20 {
		return nil
	}
	// p is memory GlobalLock pinned, not a Go pointer.
	return bytes.Clone(unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&p))), n))
}

func dropFiles(h uintptr) []string {
	n, _, _ := dragQueryFile.Call(h, 0xffffffff, 0, 0)
	var out []string
	for i := uintptr(0); i < n && i < 32; i++ {
		l, _, _ := dragQueryFile.Call(h, i, 0, 0)
		buf := make([]uint16, l+1)
		dragQueryFile.Call(h, i, uintptr(unsafe.Pointer(&buf[0])), l+1)
		out = append(out, windows.UTF16ToString(buf))
	}
	return out
}
