//go:build darwin

package native

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework ScreenCaptureKit -framework AVFoundation -framework CoreMedia -framework CoreVideo -framework Foundation
#include <stdlib.h>
#include "capture.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

// SegmentClosed is invoked from Objective-C when a rolling segment file is finalized.
var SegmentClosed func(path string)

var segmentHookMu sync.Mutex

//export shadowplayOnSegmentClosed
func shadowplayOnSegmentClosed(cpath *C.char) {
	if cpath == nil {
		return
	}
	p := C.GoString(cpath)
	segmentHookMu.Lock()
	fn := SegmentClosed
	segmentHookMu.Unlock()
	if fn != nil {
		fn(p)
	}
}

func SetSegmentClosedHook(fn func(path string)) {
	segmentHookMu.Lock()
	SegmentClosed = fn
	segmentHookMu.Unlock()
}

func StartRecording(path string) error {
	cs := C.CString(path)
	defer C.free(unsafe.Pointer(cs))
	if rc := C.sp_capture_start(cs); rc != 0 {
		return fmt.Errorf("native start failed (%d)", int(rc))
	}
	return nil
}

func StopRecording() error {
	if rc := C.sp_capture_stop(); rc != 0 {
		return fmt.Errorf("native stop failed (%d)", int(rc))
	}
	return nil
}

func IsRecording() bool {
	return C.sp_capture_is_recording() != 0
}

func RollingStart(dir string, segmentSec, maxBufferSec float64) error {
	cs := C.CString(dir)
	defer C.free(unsafe.Pointer(cs))
	if rc := C.sp_rolling_start(cs, C.double(segmentSec), C.double(maxBufferSec)); rc != 0 {
		return fmt.Errorf("rolling start failed (%d)", int(rc))
	}
	return nil
}

func RollingStop() error {
	if rc := C.sp_rolling_stop(); rc != 0 {
		return fmt.Errorf("rolling stop failed (%d)", int(rc))
	}
	return nil
}

func RollingActive() bool {
	return C.sp_rolling_is_active() != 0
}

func ExportLast(path string, durationSec float64) error {
	cs := C.CString(path)
	defer C.free(unsafe.Pointer(cs))
	if rc := C.sp_rolling_export_last(cs, C.double(durationSec)); rc != 0 {
		return fmt.Errorf("export failed (%d)", int(rc))
	}
	return nil
}
