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
