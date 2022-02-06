//go:build windows

package main

import (
	"log"
	"strings"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

var (
	vkKeyScan *windows.LazyProc
)

func init() {
	var libuser32 = windows.NewLazySystemDLL("user32.dll")
	vkKeyScan = libuser32.NewProc("VkKeyScanW")
}

func VkKeyScan(ch uint16) uint16 {
	ret, _, _ := syscall.Syscall(vkKeyScan.Addr(), 1, uintptr(ch), 0, 0)
	return uint16(ret)
}

func mouseButtonFlag(button int, press bool) uint32 {
	if button == 1 {
		if press {
			return win.MOUSEEVENTF_MIDDLEDOWN
		} else {
			return win.MOUSEEVENTF_MIDDLEUP
		}
	} else if button == 2 {
		if press {
			return win.MOUSEEVENTF_RIGHTDOWN
		} else {
			return win.MOUSEEVENTF_RIGHTUP
		}
	} else {
		if press {
			return win.MOUSEEVENTF_LEFTDOWN
		} else {
			return win.MOUSEEVENTF_LEFTUP
		}
	}
}

func move(x, y float64) {
	// TODO
	sw := win.GetSystemMetrics(win.SM_CXSCREEN)
	sh := win.GetSystemMetrics(win.SM_CYSCREEN)
	if *defaultDisplay == -1 {
		sw = win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN)
		sh = win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN)
	}
	sx := int32(x * float64(sw))
	sy := int32(y * float64(sh))
	win.SetCursorPos(sx, sy)
}

func moveW(x, y float64, hWnd uint64) bool {
	var rect win.RECT
	if !win.GetWindowRect(win.HWND(hWnd), &rect) {
		return false
	}
	win.SetForegroundWindow(win.HWND(hWnd)) // TODO
	sx := rect.Left + int32(x*float64(rect.Right-rect.Left))
	sy := rect.Top + int32(y*float64(rect.Bottom-rect.Top))
	win.SetCursorPos(sx, sy)
	return true
}

func click(button int) {
	inputs := []win.MOUSE_INPUT{
		{
			Type: win.INPUT_MOUSE,
			Mi: win.MOUSEINPUT{
				DwFlags: mouseButtonFlag(button, true),
			},
		},
		{
			Type: win.INPUT_MOUSE,
			Mi: win.MOUSEINPUT{
				DwFlags: mouseButtonFlag(button, false),
			},
		},
	}
	win.SendInput(uint32(len(inputs)), unsafe.Pointer(&inputs[0]), int32(unsafe.Sizeof(inputs[0])))
}
func dblclick(button int) {
	click(button)
	click(button)
}

func buttonState(button int, press bool) {
	inputs := []win.MOUSE_INPUT{
		{
			Type: win.INPUT_MOUSE,
			Mi: win.MOUSEINPUT{
				DwFlags: mouseButtonFlag(button, press),
			},
		},
	}
	win.SendInput(uint32(len(inputs)), unsafe.Pointer(&inputs[0]), int32(unsafe.Sizeof(inputs[0])))
}

var vkeys = map[string]uint16{
	"CONTROL": win.VK_CONTROL, "SHIFT": win.VK_LSHIFT, "ALT": win.VK_MENU, "Meta": win.VK_LWIN,
	"BACKSPACE": win.VK_BACK, "TAB": win.VK_TAB, "ENTER": win.VK_RETURN, "ESCAPE": win.VK_ESCAPE,
	"HOME": win.VK_HOME, "END": win.VK_END, "DELETE": win.VK_DELETE,
	"ARROWLEFT": win.VK_LEFT, "ARROWUP": win.VK_UP, "ARROWRIGHT": win.VK_RIGHT, "ARROWDOWN": win.VK_DOWN,
	"F1": win.VK_F1, "F2": win.VK_F2, "F3": win.VK_F3, "F4": win.VK_F4, "F5": win.VK_F5,
	"F6": win.VK_F6, "F7": win.VK_F7, "F8": win.VK_F8, "F9": win.VK_F9, "F10": win.VK_F10,
	"F11": win.VK_F11, "F12": win.VK_F12, "F13": win.VK_F13, "F14": win.VK_F14, "F15": win.VK_F15,
	"KANAMODE": win.VK_KANA,
}

type keyseq []win.KEYBD_INPUT

func (s *keyseq) KeyInput(key string) bool {
	var vk uint16
	if v, ok := vkeys[strings.ToUpper(key)]; ok {
		vk = v
	} else if len(key) == 1 {
		vk = VkKeyScan(uint16(key[0]))
		if vk == 0xffff {
			s.unicode(uint16(key[0]))
			return true
		}
	} else if len(key) != len([]rune(key)) {
		for _, c := range key {
			s.unicode(uint16(c))
		}
		return true
	} else {
		log.Println("Unsupported Key: ", key)
		return false
	}

	if vk&0x100 != 0 {
		s.vk(win.VK_LSHIFT, true)
		s.vk(vk&0xff, true)
		s.vk(vk&0xff, false)
		s.vk(win.VK_LSHIFT, false)
	} else {
		s.vk(vk&0xff, true)
		s.vk(vk&0xff, false)
	}
	return true
}

func (s *keyseq) KeyState(key string, press bool) bool {
	var vk uint16
	if v, ok := vkeys[strings.ToUpper(key)]; ok {
		vk = v
	} else if len(key) == 1 {
		scan := VkKeyScan(uint16(key[0]))
		if scan == 0xffff {
			return false
		}
		vk = scan & 0xff
	} else {
		log.Println("Unsupported Key: ", key)
		return false
	}
	s.vk(vk, press)
	return true
}

func (s *keyseq) vk(vk uint16, press bool) {
	var flags uint32 = 0
	if !press {
		flags = win.KEYEVENTF_KEYUP
	}
	*s = append(*s, win.KEYBD_INPUT{
		Type: win.INPUT_KEYBOARD,
		Ki: win.KEYBDINPUT{
			WVk:     vk,
			DwFlags: flags,
		},
	})
}

func (s *keyseq) unicode(c uint16) {
	*s = append(*s,
		win.KEYBD_INPUT{
			Type: win.INPUT_KEYBOARD,
			Ki: win.KEYBDINPUT{
				WScan:   c,
				DwFlags: win.KEYEVENTF_UNICODE,
			},
		}, win.KEYBD_INPUT{
			Type: win.INPUT_KEYBOARD,
			Ki: win.KEYBDINPUT{
				WScan:   c,
				DwFlags: win.KEYEVENTF_UNICODE | win.KEYEVENTF_KEYUP,
			},
		})
}

func (s *keyseq) Send() {
	win.SendInput(uint32(len(*s)), unsafe.Pointer(&(*s)[0]), int32(unsafe.Sizeof((*s)[0])))
}

func key(key string, metaKeys int) {
	var s keyseq
	if s.KeyInput(key) {
		s.Send()
	}
}

func keyState(key string, press bool) {
	var s keyseq
	if s.KeyState(key, press) {
		s.Send()
	}
}
