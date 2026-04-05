//go:build darwin

package native

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include "gui.h"
*/
import "C"

import (
	"log"
	"sync"
)

var (
	guiCB   GUICallbacks
	guiCBMu sync.Mutex
)

// RunGUI sets up the menu bar status item. It must be called after
// mainthread.Init has started the run loop. Blocks forever.
func RunGUI(cb GUICallbacks) {
	guiCBMu.Lock()
	guiCB = cb
	guiCBMu.Unlock()

	// Worker goroutine: sync onto the AppKit main thread (same as [NSApp run]).
	C.sp_gui_install_status_item_sync()
	log.Println("ShadowPlay menu bar ready — look for the record icon in the menu bar (may be under the chevron if the bar is full)")
	select {}
}

// GUISetBuffering updates the menu state from Go.
func GUISetBuffering(active bool) {
	v := C.int(0)
	if active {
		v = 1
	}
	C.sp_gui_set_buffering(v)
}

// GUIQuit terminates the NSApp run loop.
func GUIQuit() {
	C.sp_gui_quit()
}

func getGUICB() GUICallbacks {
	guiCBMu.Lock()
	defer guiCBMu.Unlock()
	return guiCB
}

//export shadowplayGuiOnStartBuffer
func shadowplayGuiOnStartBuffer() {
	if fn := getGUICB().OnStartBuffer; fn != nil {
		go fn()
	}
}

//export shadowplayGuiOnStopBuffer
func shadowplayGuiOnStopBuffer() {
	if fn := getGUICB().OnStopBuffer; fn != nil {
		go fn()
	}
}

//export shadowplayGuiOnSaveClip
func shadowplayGuiOnSaveClip() {
	if fn := getGUICB().OnSaveClip; fn != nil {
		go fn()
	}
}

//export shadowplayGuiOnOpenFolder
func shadowplayGuiOnOpenFolder() {
	if fn := getGUICB().OnOpenFolder; fn != nil {
		go fn()
	}
}

//export shadowplayGuiOnQuit
func shadowplayGuiOnQuit() {
	if fn := getGUICB().OnQuit; fn != nil {
		go fn()
	}
}
