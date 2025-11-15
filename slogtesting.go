// Package slogtesting is a [slog.Handler] implementation and a set of simple
// correctness checks on your application's structured logging outputs using the
// [log/slog] package. The slog.Handler outputs in-memory golang data
// structures, removing the need for your test to parse the log.
package slogtesting

import (
	"fmt"
	"log/slog"
	"slices"
)

const logPrefix = "slogtesting: "

// CaptureRecords uses the input opts to create a new handler via [NewAttrHandler],
// runs your test and collects the records processed by the handler.
//
// If opts is non-empty then its [slog.HandlerOptions] may be used to configure
// the handler, and if its CaptureRecord field is non-empty then it would be
// called after building up the output records. The run function is passed the
// handler, which should be used to create a logger via [slog.New], and then
// execute the code to test. This function may also return an error from the
// input run function. The output records are what was written to the log. See
// the package doc for an example.
func CaptureRecords(opts *AttrHandlerOptions, run func(h slog.Handler) error) (out []slog.Record, err error) {
	if opts == nil {
		opts = &AttrHandlerOptions{}
	}

	captureRecord := func(r slog.Record) (captErr error) {
		out = append(out, r)
		if opts.CaptureRecord != nil {
			captErr = opts.CaptureRecord(r)
		}
		return
	}

	handler := NewAttrHandler(&AttrHandlerOptions{
		HandlerOptions: opts.HandlerOptions,
		CaptureRecord:  captureRecord,
	})

	err = run(handler)
	return
}

// GetRecordAttrs collects each attribute on the record.
func GetRecordAttrs(r slog.Record) []slog.Attr {
	out := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		out = append(out, a)
		return true
	})
	return out
}

func collectMatchingAttrs(attrs []slog.Attr, match matcher) []slog.Attr {
	out := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		if match(attr) {
			out = append(out, attr)
		}
	}

	return slices.Clip(out)
}

func collectNMatchingAttrs(attrs []slog.Attr, n int, match matcher) (out []slog.Attr, err error) {
	out = collectMatchingAttrs(attrs, match)
	if len(out) != n {
		err = fmt.Errorf("unexpected number of matches; got %d, expected %d", len(out), n)
	}
	return
}

type matcher func(slog.Attr) bool

func makeKeyMatcher(key string) matcher { return func(a slog.Attr) bool { return a.Key == key } }
