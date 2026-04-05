//go:build darwin && !cgo

package native

import (
	"errors"
	"log"
)

// Stubs for darwin when CGO is disabled (e.g. gopls/IDE default). Real
// implementation is in native_darwin.go and gui_darwin.go with CGO_ENABLED=1.

var errNoCgo = errors.New("shadowplay: enable CGO (CGO_ENABLED=1) for ScreenCaptureKit on macOS")

func SetSegmentClosedHook(func(string)) {}

func StartRecording(string) error { return errNoCgo }
func StopRecording() error        { return errNoCgo }
func IsRecording() bool           { return false }

func RollingStart(string, float64, float64) error { return errNoCgo }
func RollingStop() error                          { return errNoCgo }
func RollingActive() bool                         { return false }
func ExportLast(string, float64) error            { return errNoCgo }

func RunGUI(GUICallbacks) {
	log.Fatal(errNoCgo)
}

func GUISetBuffering(bool) {}
func GUIQuit()             {}
