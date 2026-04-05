//go:build darwin

package hotkeyparse

import (
	"fmt"
	"strings"

	"golang.design/x/hotkey"
)

// Parse parses strings like "cmd+shift+s", "ctrl+alt+f12".
// Modifiers: cmd, command, super, shift, ctrl, control, alt, option, opt.
// Keys: a–z, 0–9, f1–f20, space, return, enter, tab, esc, escape, left, right, up, down.
func Parse(s string) (mods []hotkey.Modifier, key hotkey.Key, err error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return nil, 0, fmt.Errorf("empty hotkey")
	}
	parts := strings.Split(s, "+")
	if len(parts) < 2 {
		return nil, 0, fmt.Errorf("hotkey must include a key and at least one modifier (e.g. cmd+shift+s)")
	}
	var modSet []hotkey.Modifier
	for i := 0; i < len(parts)-1; i++ {
		p := strings.TrimSpace(parts[i])
		switch p {
		case "cmd", "command", "super":
			modSet = append(modSet, hotkey.ModCmd)
		case "shift":
			modSet = append(modSet, hotkey.ModShift)
		case "ctrl", "control":
			modSet = append(modSet, hotkey.ModCtrl)
		case "alt", "option", "opt":
			modSet = append(modSet, hotkey.ModOption)
		default:
			return nil, 0, fmt.Errorf("unknown modifier %q", p)
		}
	}
	last := strings.TrimSpace(parts[len(parts)-1])
	k, ok := keyFromToken(last)
	if !ok {
		return nil, 0, fmt.Errorf("unknown key %q", last)
	}
	return modSet, k, nil
}

func keyFromToken(t string) (hotkey.Key, bool) {
	if len(t) == 1 {
		switch t[0] {
		case 'a':
			return hotkey.KeyA, true
		case 'b':
			return hotkey.KeyB, true
		case 'c':
			return hotkey.KeyC, true
		case 'd':
			return hotkey.KeyD, true
		case 'e':
			return hotkey.KeyE, true
		case 'f':
			return hotkey.KeyF, true
		case 'g':
			return hotkey.KeyG, true
		case 'h':
			return hotkey.KeyH, true
		case 'i':
			return hotkey.KeyI, true
		case 'j':
			return hotkey.KeyJ, true
		case 'k':
			return hotkey.KeyK, true
		case 'l':
			return hotkey.KeyL, true
		case 'm':
			return hotkey.KeyM, true
		case 'n':
			return hotkey.KeyN, true
		case 'o':
			return hotkey.KeyO, true
		case 'p':
			return hotkey.KeyP, true
		case 'q':
			return hotkey.KeyQ, true
		case 'r':
			return hotkey.KeyR, true
		case 's':
			return hotkey.KeyS, true
		case 't':
			return hotkey.KeyT, true
		case 'u':
			return hotkey.KeyU, true
		case 'v':
			return hotkey.KeyV, true
		case 'w':
			return hotkey.KeyW, true
		case 'x':
			return hotkey.KeyX, true
		case 'y':
			return hotkey.KeyY, true
		case 'z':
			return hotkey.KeyZ, true
		case '0':
			return hotkey.Key0, true
		case '1':
			return hotkey.Key1, true
		case '2':
			return hotkey.Key2, true
		case '3':
			return hotkey.Key3, true
		case '4':
			return hotkey.Key4, true
		case '5':
			return hotkey.Key5, true
		case '6':
			return hotkey.Key6, true
		case '7':
			return hotkey.Key7, true
		case '8':
			return hotkey.Key8, true
		case '9':
			return hotkey.Key9, true
		}
	}

	switch t {
	case "space":
		return hotkey.KeySpace, true
	case "return", "enter":
		return hotkey.KeyReturn, true
	case "tab":
		return hotkey.KeyTab, true
	case "esc", "escape":
		return hotkey.KeyEscape, true
	case "delete", "backspace":
		return hotkey.KeyDelete, true
	case "left":
		return hotkey.KeyLeft, true
	case "right":
		return hotkey.KeyRight, true
	case "up":
		return hotkey.KeyUp, true
	case "down":
		return hotkey.KeyDown, true
	case "f1":
		return hotkey.KeyF1, true
	case "f2":
		return hotkey.KeyF2, true
	case "f3":
		return hotkey.KeyF3, true
	case "f4":
		return hotkey.KeyF4, true
	case "f5":
		return hotkey.KeyF5, true
	case "f6":
		return hotkey.KeyF6, true
	case "f7":
		return hotkey.KeyF7, true
	case "f8":
		return hotkey.KeyF8, true
	case "f9":
		return hotkey.KeyF9, true
	case "f10":
		return hotkey.KeyF10, true
	case "f11":
		return hotkey.KeyF11, true
	case "f12":
		return hotkey.KeyF12, true
	case "f13":
		return hotkey.KeyF13, true
	case "f14":
		return hotkey.KeyF14, true
	case "f15":
		return hotkey.KeyF15, true
	case "f16":
		return hotkey.KeyF16, true
	case "f17":
		return hotkey.KeyF17, true
	case "f18":
		return hotkey.KeyF18, true
	case "f19":
		return hotkey.KeyF19, true
	case "f20":
		return hotkey.KeyF20, true
	default:
		return 0, false
	}
}

// SameBinding reports whether two hotkey strings parse to the same shortcut.
func SameBinding(a, b string) bool {
	ma, ka, ea := Parse(a)
	mb, kb, eb := Parse(b)
	if ea != nil || eb != nil {
		return false
	}
	if ka != kb || len(ma) != len(mb) {
		return false
	}
	var sumA, sumB hotkey.Modifier
	for _, m := range ma {
		sumA += m
	}
	for _, m := range mb {
		sumB += m
	}
	return sumA == sumB
}
