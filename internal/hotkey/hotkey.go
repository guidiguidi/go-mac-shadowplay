package hotkey

import (
	"context"
	"fmt"

	hook "github.com/robotn/gohook"
)

type HotkeyHandler struct {
	onToggle func()
	onSave   func()
}

func NewHotkeyHandler(onToggle, onSave func()) *HotkeyHandler {
	return &HotkeyHandler{
		onToggle: onToggle,
		onSave:   onSave,
	}
}

func (h *HotkeyHandler) Listen(ctx context.Context) {
	fmt.Println("🔍 Using robotgo hooks for keyboard detection...")
	fmt.Println("⌨️  Hotkeys: Opt+Shift+R (Toggle), Opt+Shift+S (Save)")
	
	evChan := hook.Start()
	defer hook.End()

	for {
		select {
		case ev := <-evChan:
			// Check for Shift (mask 1) + Opt (mask 8)
			// robotgo hook masks: shift=1, ctrl=2, alt/opt=8, cmd=16
			isShift := ev.Mask & 1 != 0
			isOpt := ev.Mask & 8 != 0

			if isShift && isOpt {
				if ev.Kind == hook.KeyDown {
					if ev.Rawcode == 15 { // R key on Mac
						fmt.Println("[HOOK] Opt+Shift+R detected")
						h.onToggle()
					} else if ev.Rawcode == 1 { // S key on Mac
						fmt.Println("[HOOK] Opt+Shift+S detected")
						h.onSave()
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
