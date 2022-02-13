//go:build windows

package main

import (
	"log"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	vkKeyScan              = user32.NewProc("VkKeyScanW")
	enumDisplayDevicesW    = user32.NewProc("EnumDisplayDevicesW")
	enumDisplayMonitors    = user32.NewProc("EnumDisplayMonitors")
	enumDisplaySettingsExW = user32.NewProc("EnumDisplaySettingsExW")
	setProcessDPIAware     = user32.NewProc("SetProcessDPIAware")
)

func init() {
	setProcessDpiAwarenessContext := user32.NewProc("SetProcessDpiAwarenessContext")
	if err := setProcessDpiAwarenessContext.Find(); err == nil {
		var DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 int32 = -4
		r0, _, _ := setProcessDpiAwarenessContext.Call(uintptr(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2))
		if r0 != 1 {
			return
		}
	} else {
		syscall.Syscall(setProcessDPIAware.Addr(), 0, 0, 0, 0)
		//SetProcessDpiAwareness(2)
	}
}

func VkKeyScan(ch uint16) uint16 {
	ret, _, _ := syscall.Syscall(vkKeyScan.Addr(), 1, uintptr(ch), 0, 0)
	return uint16(ret)
}

type MonitorCallback func(hMonitor windows.Handle, hdcMonitor windows.Handle, lprcMonitor *win.RECT) int

type DISPLAY_DEVICE struct {
	Size         uint32
	DeviceName   [32]uint16
	DeviceString [128]uint16
	Flags        uint32
	DeviceID     [128]uint16
	DeviceKey    [128]uint16
}

type DEVMODEW struct {
	DeviceName    [32]uint16
	SpecVersion   uint16
	DriverVersion uint16
	Size          uint16
	DriverExtra   uint16
	Fields        uint32
	X             int32
	Y             int32
	Orientation   uint32
	FixedOutput   uint32
	Color         int16
	Duplex        int16
	YResolution   int16
	TTOption      int16
	Collate       int16
	FormName      [32]uint16
	LogPixels     uint16
	BitsPerPixel  uint32
	PelsWidth     uint32
	PelsHeight    uint32
	Flags         uint32
	Frequency     uint32
	ICMMethod     uint32
	ICMIntent     uint32
	MediaType     uint32
	DitherType    uint32
	Reserved1     uint32
	Reserved2     uint32
	PanningWidth  uint32
	PanningHeight uint32
}

func EnumDisplayDevices(dummy string, idx int, displayDevice *DISPLAY_DEVICE, flags uint32) bool {
	displayDevice.Size = uint32(unsafe.Sizeof(*displayDevice))
	ret, _, _ := syscall.Syscall6(enumDisplayDevicesW.Addr(), 4, 0, uintptr(idx), uintptr(unsafe.Pointer(displayDevice)), uintptr(flags), 0, 0)
	return ret != 0
}

func EnumDisplaySettingsEx(deviceName []uint16, modeNum int32, devMode *DEVMODEW, flags uint32) bool {
	// ENUM_CURRENT_SETTINGS = -1
	devMode.Size = uint16(unsafe.Sizeof(*devMode))
	ret, _, _ := enumDisplaySettingsExW.Call(uintptr(unsafe.Pointer(&deviceName[0])), uintptr(modeNum), uintptr(unsafe.Pointer(devMode)), uintptr(flags))
	return ret != 0
}

func GetMonitorsRect() []win.RECT {
	var dd DISPLAY_DEVICE
	var ret []win.RECT
	for i := 0; EnumDisplayDevices("", i, &dd, 0); i++ {
		var mode DEVMODEW
		if dd.Flags&1 != 0 && EnumDisplaySettingsEx(dd.DeviceName[:], -1, &mode, 0) {
			rect := win.RECT{Left: mode.X, Top: mode.Y, Right: mode.X + int32(mode.PelsWidth), Bottom: mode.Y + int32(mode.PelsHeight)}
			ret = append(ret, rect)
		} else {
			var rect win.RECT
			rect.Right = win.GetSystemMetrics(win.SM_CXSCREEN)
			rect.Bottom = win.GetSystemMetrics(win.SM_CYSCREEN)
			ret = append(ret, rect)
		}
	}
	return ret
}

var getMonitorBoundsCallback = syscall.NewCallback(func(hMonitor windows.Handle, hdcMonitor windows.Handle, lprcMonitor *win.RECT, dwData uintptr) uintptr {
	if dwData == 0 {
		return uintptr(0)
	}
	var cb *MonitorCallback = nil
	cb = (*MonitorCallback)(unsafe.Pointer(uintptr(unsafe.Pointer(cb)) + dwData))
	return uintptr((*cb)(hMonitor, hdcMonitor, lprcMonitor))
})

func EnumDisplayMonitors(cb MonitorCallback) bool {
	ret, _, _ := syscall.Syscall6(enumDisplayMonitors.Addr(), 4, 0, 0, getMonitorBoundsCallback, uintptr(unsafe.Pointer(&cb)), 0, 0)
	return ret != 0
}

func GetMonitorsRect2() []win.RECT {
	var ret []win.RECT
	EnumDisplayMonitors(func(hMonitor windows.Handle, hdcMonitor windows.Handle, lprcMonitor *win.RECT) int {
		log.Println(hMonitor, hdcMonitor)
		ret = append(ret, *lprcMonitor)
		return 1
	})
	return ret
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

var lastUpdate time.Time
var monitors []win.RECT
var monitorMutex sync.Mutex

func moveD(x, y float64, display uint64) bool {
	monitorMutex.Lock()
	defer monitorMutex.Unlock()
	now := time.Now()
	if now.Sub(lastUpdate).Seconds() > 10 {
		monitors = GetMonitorsRect()
		lastUpdate = now
		log.Printf("monitors: %#v", monitors)
	}
	if display >= uint64(len(monitors)) {
		return false
	}
	rect := monitors[int(display)]
	sx := rect.Left + int32(x*float64(rect.Right-rect.Left))
	sy := rect.Top + int32(y*float64(rect.Bottom-rect.Top))
	return win.SetCursorPos(sx, sy)
}

func moveW(x, y float64, hWnd uint64) bool {
	var rect win.RECT
	if !win.GetWindowRect(win.HWND(hWnd), &rect) {
		return false
	}
	win.SetForegroundWindow(win.HWND(hWnd)) // TODO
	sx := rect.Left + int32(x*float64(rect.Right-rect.Left))
	sy := rect.Top + int32(y*float64(rect.Bottom-rect.Top))
	return win.SetCursorPos(sx, sy)
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
	"HOME": win.VK_HOME, "END": win.VK_END, "DELETE": win.VK_DELETE, "CAPSLOCK": win.VK_CAPITAL,
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
