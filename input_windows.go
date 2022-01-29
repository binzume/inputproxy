//go:build windows

package main

import (
	"strings"
	"unsafe"

	"github.com/lxn/win"
)

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

func key(key string, metaKeys int) {
	keyState(key, true)
	keyState(key, false)
}

var vkeys = map[string]uint16{
	"CTRL": win.VK_CONTROL, "SHIFT": win.VK_LSHIFT, "ALT": win.VK_MENU,
	"BACK": win.VK_BACK, "TAB": win.VK_TAB, "ENTER": win.VK_RETURN, "ESC": win.VK_ESCAPE,
	"F1": win.VK_F1, "F2": win.VK_F2, "F3": win.VK_F3, "F4": win.VK_F4, "F5": win.VK_F5,
	"F6": win.VK_F6, "F7": win.VK_F7, "F8": win.VK_F8, "F9": win.VK_F9, "F10": win.VK_F10,
}

func keyState(key string, press bool) {
	var flags uint32 = 0
	if !press {
		flags = win.KEYEVENTF_KEYUP
	}
	key = strings.ToUpper(key)
	var vk uint16
	if v, ok := vkeys[key]; ok {
		vk = v
	} else {
		vk = uint16(key[0])
	}
	inputs := []win.KEYBD_INPUT{
		{
			Type: win.INPUT_KEYBOARD,
			Ki: win.KEYBDINPUT{
				WVk:     uint16(vk),
				DwFlags: flags,
			},
		},
	}
	win.SendInput(uint32(len(inputs)), unsafe.Pointer(&inputs[0]), int32(unsafe.Sizeof(inputs[0])))
}
