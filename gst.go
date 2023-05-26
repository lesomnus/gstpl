package gstpl

/*
#cgo pkg-config: gstreamer-1.0
#include "gst.h"
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"
)

func gerror(gerr *C.GError) error {
	if gerr == nil {
		return nil
	}

	defer C.g_error_free(gerr)
	msg := C.GoString(gerr.message)

	return fmt.Errorf("%s (%d)", msg, int(gerr.code))
}

//export goHandleEndOfStream
func goHandleEndOfStream(handler unsafe.Pointer) {
	pl := (*pipeline)(handler)
	pl.cancel()
}

//export goHandleSample
func goHandleSample(handler unsafe.Pointer, buff unsafe.Pointer, len C.int, duration C.int) {
	pl := (*pipeline)(handler)
	sample := Sample{Data: C.GoBytes(buff, len), Duration: time.Duration(duration)}

	select {
	case <-pl.ctx.Done():
	case pl.samples <- sample:
	}
}

//export goHandleError
func goHandleError(handler unsafe.Pointer, gerr *C.GError) {
	// It is possible to miss the last error, but I don't care about it that much.

	pl := (*pipeline)(handler)
	pl.err.Store(gerror(gerr))
	select {
	case pl.got_err <- struct{}{}:
	default:
	}
}

func loopCtxRefCnt() int {
	return int(C.gstpl_ref_cnt())
}

func init() {
	var gerr *C.GError
	if !C.gstpl_init(&gerr) {
		panic(gerror(gerr))
	}
}
