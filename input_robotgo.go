//go:build !windows

package main

import (
	"github.com/go-vgo/robotgo"
	"time"
)

func buttonName(button int) string {
	switch button {
	case 0:
		return "left"
	case 1:
		return "middle"
	case 2:
		return "right"
	default:
		return "left"
	}
}

func click(button int) {
	robotgo.Click(buttonName(button), false)
}

func dblclick(button int) {
	robotgo.Click(buttonName(button), true)
}

func move(x, y float64) {
	sw, sh := robotgo.GetScreenSize()
	sx := int(x * float64(sw))
	sy := int(y * float64(sh))
	robotgo.Move(sx, sy)
	time.Sleep(time.Millisecond)
}

func moveW(x, y float64, windowId uint64) bool {
	return false
}

func buttonState(button int, press bool) {
	var state string
	if press {
		state = "down"
	} else {
		state = "up"
	}
	robotgo.Toggle(buttonName(button), state)
}

func key(key string, metaKeys int) {
	robotgo.KeyTap(key)
}

func keyState(key string, press bool) {
	if press {
		robotgo.KeyDown(key)
	} else {
		robotgo.KeyUp(key)
	}
}
