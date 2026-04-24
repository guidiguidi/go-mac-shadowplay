//go:build darwin

package native

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework UserNotifications
#include <stdlib.h>
#include "gui.h"
*/
import "C"

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/guidiguidi/go-mac-shadowplay/internal/config"
)

var (
	guiCB   GUICallbacks
	guiCBMu sync.Mutex

	prefsCfg     config.Config
	prefsPath    string
	prefsStateMu sync.Mutex
)

// prefsDTO matches JSON keys produced by the ObjC preferences panel.
type prefsDTO struct {
	BufferMinutes  int     `json:"buffer_minutes"`
	ClipSeconds    int     `json:"clip_seconds"`
	SegmentSeconds float64 `json:"segment_seconds"`
	OutputDir      string  `json:"output_dir"`
	TempDir        string  `json:"temp_dir"`
	SaveHotkey     string  `json:"save_hotkey"`
	RecordHotkey   string  `json:"record_hotkey"`
}

func resolveGUIConfigPath(p string) string {
	if p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "shadowplay-config.yaml")
	}
	return filepath.Join(home, ".config", "shadowplay", "config.yaml")
}

// RunGUI starts the menu bar UI. configPath is where YAML is saved after editing
// (empty string uses ~/.config/shadowplay/config.yaml).
func RunGUI(configPath string, initial config.Config, cb GUICallbacks) {
	guiCBMu.Lock()
	guiCB = cb
	guiCBMu.Unlock()

	prefsStateMu.Lock()
	prefsPath = resolveGUIConfigPath(configPath)
	prefsCfg = initial
	prefsStateMu.Unlock()

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

// GUINotify shows a macOS system notification.
func GUINotify(title, message string) {
	ct := C.CString(title)
	cm := C.CString(message)
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cm))
	C.sp_gui_notify(ct, cm)
}

func getGUICB() GUICallbacks {
	guiCBMu.Lock()
	defer guiCBMu.Unlock()
	return guiCB
}

func dtoFromConfig(c config.Config) prefsDTO {
	return prefsDTO{
		BufferMinutes:  c.BufferMinutes,
		ClipSeconds:    c.ClipSeconds,
		SegmentSeconds: c.SegmentSeconds,
		OutputDir:      c.OutputDir,
		TempDir:        c.TempDir,
		SaveHotkey:     c.SaveHotkey,
		RecordHotkey:   c.RecordHotkey,
	}
}

func configFromDTO(d prefsDTO) config.Config {
	c := config.Default()
	c.BufferMinutes = d.BufferMinutes
	c.ClipSeconds = d.ClipSeconds
	c.SegmentSeconds = d.SegmentSeconds
	c.OutputDir = d.OutputDir
	c.TempDir = d.TempDir
	c.SaveHotkey = d.SaveHotkey
	c.RecordHotkey = d.RecordHotkey
	if c.BufferMinutes <= 0 {
		c.BufferMinutes = 10
	}
	if c.ClipSeconds <= 0 {
		c.ClipSeconds = 30
	}
	if c.SegmentSeconds <= 0 {
		c.SegmentSeconds = 3
	}
	c.OutputDir = configExpandForGUI(c.OutputDir)
	c.TempDir = configExpandForGUI(c.TempDir)
	return c
}

// Duplicates config.expandPath logic without exporting it.
func configExpandForGUI(s string) string {
	if len(s) >= 2 && s[0] == '~' && (s[1] == '/' || s[1] == '\\') {
		home, err := os.UserHomeDir()
		if err != nil {
			return s
		}
		return filepath.Join(home, s[2:])
	}
	return s
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

//export shadowplayGuiOnPreferences
func shadowplayGuiOnPreferences() {
	go func() {
		cb := getGUICB()
		if cb.IsBufferActive != nil && cb.IsBufferActive() {
			t := C.CString("ShadowPlay")
			m := C.CString("Stop the buffer before changing settings (hotkeys and segment size need a restart).")
			C.sp_gui_alert(t, m)
			C.free(unsafe.Pointer(t))
			C.free(unsafe.Pointer(m))
			return
		}

		prefsStateMu.Lock()
		cur := prefsCfg
		path := prefsPath
		prefsStateMu.Unlock()

		dto := dtoFromConfig(cur)
		jsonBytes, err := json.Marshal(dto)
		if err != nil {
			log.Println("prefs json:", err)
			return
		}

		cin := C.CString(string(jsonBytes))
		defer C.free(unsafe.Pointer(cin))
		var cout *C.char
		rc := C.sp_gui_prefs_modal(cin, &cout)
		if rc == 0 || cout == nil {
			return
		}
		defer C.free(unsafe.Pointer(cout))

		var out prefsDTO
		if err := json.Unmarshal([]byte(C.GoString(cout)), &out); err != nil {
			log.Println("prefs parse:", err)
			return
		}
		newCfg := configFromDTO(out)

		if err := config.Save(path, newCfg); err != nil {
			log.Println("save config:", err)
			t := C.CString("Could not save settings")
			msg := C.CString(err.Error())
			C.sp_gui_alert(t, msg)
			C.free(unsafe.Pointer(t))
			C.free(unsafe.Pointer(msg))
			return
		}

		prefsStateMu.Lock()
		prefsCfg = newCfg
		prefsStateMu.Unlock()

		if cb.OnConfigSaved != nil {
			cb.OnConfigSaved(newCfg)
		}
		log.Println("saved settings to", path)
	}()
}
