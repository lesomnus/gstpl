package gstpl

/*
#cgo pkg-config: gstreamer-1.0
#include "gst.h"
*/
import "C"
import (
	"context"
	"io"
	"sync/atomic"
	"time"
	"unsafe"
)

type Sample struct {
	Data     []byte
	Duration time.Duration
}

// Pipeline represents a GStreamer pipeline instance.
type Pipeline interface {
	Start() error
	Recv() (Sample, error)
	Close() error
}

type pipeline struct {
	gst_ctx *C.Context
	got_err chan struct{}
	samples chan Sample

	ctx    context.Context
	cancel context.CancelFunc

	err atomic.Value

	has_started bool
}

// NewPipeline builds a GStreamer pipeline.
func NewPipeline(expr string) (Pipeline, error) {
	expr = expr + " ! appsink name=appsink"

	unsafe_expr := C.CString(expr)
	defer C.free(unsafe.Pointer(unsafe_expr))

	var gerr *C.GError
	gst_ctx := C.gstpl_ctx_new(unsafe_expr, &gerr)
	if gerr != nil {
		return nil, gerror(gerr)
	}

	ctx, cancel := context.WithCancel(context.Background())

	pl := &pipeline{
		gst_ctx: gst_ctx,
		got_err: make(chan struct{}, 1),
		samples: make(chan Sample),

		ctx:    ctx,
		cancel: cancel,
	}
	gst_ctx.handler = unsafe.Pointer(pl)

	return pl, nil
}

func (p *pipeline) Start() error {
	if p.gst_ctx == nil {
		return io.ErrClosedPipe
	}
	if p.has_started {
		return nil
	}

	C.gstpl_ctx_start(p.gst_ctx)
	p.has_started = true
	return nil
}

func (p *pipeline) Recv() (Sample, error) {
	select {
	case <-p.ctx.Done():
		return Sample{}, io.EOF

	case sample := <-p.samples:
		return sample, nil

	case <-p.got_err:
		return Sample{}, p.err.Load().(error)
	}
}

func (p *pipeline) Close() error {
	if p.gst_ctx == nil {
		return nil
	}

	// p.gst_ctx.handler = nil
	C.gstpl_ctx_free(p.gst_ctx)
	p.gst_ctx = nil

	p.cancel()
	return nil
}
