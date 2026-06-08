package slogtesting

import (
	"context"
	"log/slog"
	"slices"
	"sync"
)

type attrHandler struct {
	opts AttrHandlerOptions
	mtx  *sync.Mutex
	goas *groupsOrAttrs
}

// AttrHandlerOptions is a superset of [slog.HandlerOptions] for use in
// [NewAttrHandler]. The CaptureRecord field is a callback function for using a
// record processed by the handler's Handle method.
type AttrHandlerOptions struct {
	slog.HandlerOptions
	CaptureRecord func(r slog.Record) error
}

// NewAttrHandler creates a [slog.Handler] that outputs attributes without any
// kind of formatting. It's intended to help with testing: rather than parsing
// formatted log entries, look at golang data structures.
//
// Its Handle method builds up a new slog.Record and passes the result to a
// function, CaptureRecord, which is set when creating the Handler. Use
// [GetRecordAttrs] to access the attributes of the processed record. Unless the
// handler was created with a CaptureRecord function, the Handle method is a
// no-op.
//
// If all you need is to run a test involving some logging action and to inspect
// the logging ouput, then [CaptureRecords] might suit that need more directly.
// For other use cases, this handler is available.
func NewAttrHandler(opts *AttrHandlerOptions) slog.Handler {
	if opts == nil {
		opts = &AttrHandlerOptions{}
	}
	return &attrHandler{
		opts: *opts,
		mtx:  &sync.Mutex{},
	}
}

func (h *attrHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	level := slog.LevelInfo
	if h.opts.Level != nil {
		level = h.opts.Level.Level()
	}
	enabled := lvl >= level
	return enabled
}

func (h *attrHandler) Handle(_ context.Context, rec slog.Record) (err error) {
	capture := h.opts.CaptureRecord
	if capture == nil {
		return
	}

	out := h.buildRecordAttrs(rec)

	h.mtx.Lock()
	defer h.mtx.Unlock()
	err = capture(out)
	return
}

func (h *attrHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	out := *h
	out.goas = &groupsOrAttrs{Attrs: attrs, Next: h.goas}

	return &out
}

func (h *attrHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	out := *h
	out.goas = &groupsOrAttrs{Group: name, Next: h.goas}

	return &out
}

func (h *attrHandler) buildRecordAttrs(r slog.Record) slog.Record {
	out := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	ab := attrBuilder{
		replaceAttr: h.opts.ReplaceAttr,
		attrsByPath: make(map[string]*attrWithPath, max(r.NumAttrs()-3, 0)),
		results:     make([]slog.Attr, 0, r.NumAttrs()+3),
	}

	// Start with builtin attributes. These are already on the input record.

	// From slog.Handler docs:
	// 	If r.Time is the zero time, ignore the time.
	if !r.Time.IsZero() {
		ab.buildAttr(nil, slog.Time(slog.TimeKey, r.Time))
	}
	ab.buildAttr(nil, slog.String(slog.LevelKey, r.Level.String()))
	if h.opts.AddSource {
		src := r.Source()
		if src == nil {
			src = &slog.Source{}
		}
		ab.buildAttr(nil, slog.Any(slog.SourceKey, src))
	}
	ab.buildAttr(nil, slog.String(slog.MessageKey, r.Message))

	// Work on the non builtin attributes.
	// Start on data accumulated from .WithAttrs, .WithGroup.
	groups := applyGroupsOrAttrs(h.goas, ab.buildAttr)

	// Now work on attributes passed in from the logger's output method.
	r.Attrs(func(a slog.Attr) bool {
		ab.buildAttr(groups, a)
		return true
	})

	ab.results = slices.Clip(ab.results)
	out.AddAttrs(ab.results...)
	return out
}
