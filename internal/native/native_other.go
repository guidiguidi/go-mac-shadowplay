//go:build !darwin

package native

import "errors"

var ErrUnsupported = errors.New("shadowplay: supported only on macOS")

func SetSegmentClosedHook(func(string)) {}

func StartRecording(string) error { return ErrUnsupported }
func StopRecording() error        { return ErrUnsupported }
func IsRecording() bool           { return false }

func RollingStart(string, float64, float64) error { return ErrUnsupported }
func RollingStop() error                          { return ErrUnsupported }
func RollingActive() bool                         { return false }
func ExportLast(string, float64) error            { return ErrUnsupported }
