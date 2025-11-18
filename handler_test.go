package slogtesting_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"os"
	"testing"
	"testing/slogtest"
	"time"

	st "github.com/rafaelespinoza/slogtesting"
)

func TestSlogtest(t *testing.T) {
	originalDefaults := slog.Default()
	t.Cleanup(func() { slog.SetDefault(originalDefaults) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				attr = slog.Attr{}
			}
			return attr
		},
	})))

	var buf bytes.Buffer
	captureJSONRecord := makeJSONRecordCapturer(&buf)
	newHandler := func(t *testing.T) slog.Handler {
		t.Helper()
		buf.Reset() // try to make each test a clean slate.
		opts := st.AttrHandlerOptions{
			HandlerOptions: slog.HandlerOptions{Level: slog.LevelInfo},
			CaptureRecord:  captureJSONRecord,
		}
		return st.NewAttrHandler(&opts)
	}
	makeTestResults := func(t *testing.T) (out map[string]any) {
		t.Helper()
		line := buf.Bytes()
		if len(line) == 0 {
			return
		}
		t.Logf("%s", line)
		if err := json.Unmarshal(line, &out); err != nil {
			t.Fatal(err)
		}
		return
	}

	slogtest.Run(t, newHandler, makeTestResults)
}

func TestAttrHandler(t *testing.T) {
	tests := []struct {
		name   string
		opts   *st.AttrHandlerOptions
		action func(*testing.T, slog.Handler)
		expect func(*testing.T, []slog.Record)
	}{
		{
			name: "options.AddSource",
			opts: &st.AttrHandlerOptions{
				HandlerOptions: slog.HandlerOptions{AddSource: true},
			},
			action: func(t *testing.T, h slog.Handler) {
				rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 1)
				err := h.Handle(context.Background(), rec)
				if err != nil {
					t.Error(err)
				}
			},
			expect: func(t *testing.T, got []slog.Record) {
				requireResultLen(t, got, 1)
				attrs := st.GetRecordAttrs(got[0])
				check := st.HasKey(slog.SourceKey)
				if err := check(attrs); err != nil {
					t.Error(err)
				}
			},
		},
		{
			name: "options.ReplaceAttr time key",
			opts: &st.AttrHandlerOptions{
				HandlerOptions: slog.HandlerOptions{
					ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
						if a.Key == slog.TimeKey {
							a.Key = "timestamp"
						}
						return a
					},
				},
			},
			action: func(t *testing.T, h slog.Handler) {
				rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
				err := h.Handle(context.Background(), rec)
				if err != nil {
					t.Error(err)
				}
			},
			expect: func(t *testing.T, got []slog.Record) {
				requireResultLen(t, got, 1)
				attrs := st.GetRecordAttrs(got[0])
				check := st.HasKey("timestamp")
				if err := check(attrs); err != nil {
					t.Error(err)
				}
			},
		},
		{
			name: "options.ReplaceAttr level key",
			opts: &st.AttrHandlerOptions{
				HandlerOptions: slog.HandlerOptions{
					ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
						if a.Key == slog.LevelKey {
							a.Key = "sev"
						}
						return a
					},
				},
			},
			action: func(t *testing.T, h slog.Handler) {
				rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
				err := h.Handle(context.Background(), rec)
				if err != nil {
					t.Error(err)
				}
			},
			expect: func(t *testing.T, got []slog.Record) {
				requireResultLen(t, got, 1)
				attrs := st.GetRecordAttrs(got[0])
				check := st.HasKey("sev")
				if err := check(attrs); err != nil {
					t.Error(err)
				}
			},
		},
		{
			name: "initialized with empty options",
			opts: nil,
			action: func(t *testing.T, h slog.Handler) {
				rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
				err := h.Handle(context.Background(), rec)
				if err != nil {
					t.Error(err)
				}
			},
			expect: func(t *testing.T, got []slog.Record) {
				requireResultLen(t, got, 0)
			},
		},
		{
			name: "deep group",
			opts: &st.AttrHandlerOptions{},
			action: func(t *testing.T, h slog.Handler) {
				rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
				rec.AddAttrs(slog.GroupAttrs("G",
					slog.GroupAttrs("H", slog.Bool("deep", true)),
				))
				err := h.Handle(context.Background(), rec)
				if err != nil {
					t.Error(err)
				}
			},
			expect: func(t *testing.T, got []slog.Record) {
				requireResultLen(t, got, 1)
				attrs := st.GetRecordAttrs(got[0])
				check := st.InGroup("G",
					st.InGroup("H",
						st.HasAttr(slog.Bool("deep", true)),
					),
				)
				if err := check(attrs); err != nil {
					t.Error(err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got []slog.Record
			if test.opts != nil {
				test.opts.CaptureRecord = func(r slog.Record) error {
					got = append(got, r)
					printRecordAttrsJSON(t, r)
					return nil
				}
			}

			h := st.NewAttrHandler(test.opts)
			test.action(t, h)

			test.expect(t, got)
		})
	}
}

func TestAttrHandlerEnabled(t *testing.T) {
	tests := []struct {
		name              string
		handlerOptionsNil bool
		handlerLevel      slog.Level
		expDebugOutput    bool
		expInfoOutput     bool
		expWarnOutput     bool
		expErrorOutput    bool
	}{
		{
			handlerLevel:   slog.LevelDebug,
			expDebugOutput: true, expInfoOutput: true, expWarnOutput: true, expErrorOutput: true,
		},
		{
			handlerLevel:   slog.LevelInfo,
			expDebugOutput: false, expInfoOutput: true, expWarnOutput: true, expErrorOutput: true,
		},
		{
			handlerLevel:   slog.LevelWarn,
			expDebugOutput: false, expInfoOutput: false, expWarnOutput: true, expErrorOutput: true,
		},
		{
			handlerLevel:   slog.LevelError,
			expDebugOutput: false, expInfoOutput: false, expWarnOutput: false, expErrorOutput: true,
		},
		{
			handlerLevel:   slog.LevelError + 1,
			expDebugOutput: false, expInfoOutput: false, expWarnOutput: false, expErrorOutput: false,
		},
		{
			name:              "handler options nil",
			handlerOptionsNil: true,
			expDebugOutput:    false, expInfoOutput: true, expWarnOutput: true, expErrorOutput: true,
		},
	}

	runTest := func(t *testing.T, optsNil bool, handlerLevel, recordLevel slog.Level, expOutput bool) {
		var opts *st.AttrHandlerOptions
		if !optsNil {
			opts = &st.AttrHandlerOptions{
				HandlerOptions: slog.HandlerOptions{Level: handlerLevel},
			}
		}
		handler := st.NewAttrHandler(opts)
		enabled := handler.Enabled(context.Background(), recordLevel)
		if expOutput && !enabled {
			t.Errorf("expected data at level %s", recordLevel.String())
		} else if !expOutput && enabled {
			t.Errorf("unexpected data at level %s", recordLevel.String())
		}
	}

	for _, test := range tests {
		name := test.name
		if name == "" {
			name = "handler level " + test.handlerLevel.String()
		}

		t.Run(name, func(t *testing.T) {
			t.Run("record "+slog.LevelDebug.String(), func(t *testing.T) {
				runTest(t, test.handlerOptionsNil, test.handlerLevel, slog.LevelDebug, test.expDebugOutput)
			})

			t.Run("record "+slog.LevelInfo.String(), func(t *testing.T) {
				runTest(t, test.handlerOptionsNil, test.handlerLevel, slog.LevelInfo, test.expInfoOutput)
			})

			t.Run("record "+slog.LevelWarn.String(), func(t *testing.T) {
				runTest(t, test.handlerOptionsNil, test.handlerLevel, slog.LevelWarn, test.expWarnOutput)
			})

			t.Run("record "+slog.LevelError.String(), func(t *testing.T) {
				runTest(t, test.handlerOptionsNil, test.handlerLevel, slog.LevelError, test.expErrorOutput)
			})
		})
	}
}

func TestAttrHandlerNoCapture(t *testing.T) {
	// Sanity check that a panic does not occur when a Handler is initialized
	// without a record capture callback.
	defer func() {
		if got := recover(); got != nil {
			t.Errorf("handler.Handle should not panic if its records capture callback is nil %v", got)
		}
	}()

	h := st.NewAttrHandler(&st.AttrHandlerOptions{})
	err := h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0))
	if err != nil {
		t.Error(err)
	}
}

func makeJSONRecordCapturer(w io.Writer) func(r slog.Record) error {
	return func(r slog.Record) error {
		attrs := st.GetRecordAttrs(r)
		mappedAttrs := mapAttrs(attrs)
		return json.NewEncoder(w).Encode(mappedAttrs)
	}
}

func mapAttrs(attrs []slog.Attr) map[string]any {
	out := make(map[string]any, len(attrs))

	for _, attr := range attrs {
		prevVal, exists := out[attr.Key]
		if !exists {
			out[attr.Key] = slogValueToAny(attr.Value)
			continue
		}

		prevMap, isMap := prevVal.(map[string]any)
		if isMap && attr.Value.Kind() == slog.KindGroup {
			currMap := mapAttrs(attr.Value.Group())
			out[attr.Key] = mergeMaps(prevMap, currMap)
		}
	}

	return out
}

func slogValueToAny(val slog.Value) (out any) {
	switch val.Kind() {
	case slog.KindAny:
		out = val.Any()
	case slog.KindBool:
		out = val.Bool()
	case slog.KindDuration:
		out = val.Duration()
	case slog.KindFloat64:
		out = val.Float64()
	case slog.KindInt64:
		out = val.Int64()
	case slog.KindString:
		out = val.String()
	case slog.KindTime:
		out = val.Time()
	case slog.KindUint64:
		out = val.Uint64()
	case slog.KindGroup:
		out = mapAttrs(val.Group())
	case slog.KindLogValuer:
		out = val.LogValuer().LogValue()
	default:
		out = val.Any()
	}
	return
}

func mergeMaps(prev, next map[string]any) map[string]any {
	if len(prev) > 0 && len(next) < 1 {
		return prev
	} else if len(prev) < 1 && len(next) > 0 {
		return next
	} else if len(prev) < 1 && len(next) < 1 {
		return make(map[string]any, 0)
	}

	maxLength := max(len(prev), len(next))
	out := make(map[string]any, maxLength)

	for k := range prev {
		out[k] = prev[k]
	}

	for nextKey, nextVal := range next {
		prevVal, found := out[nextKey]
		if !found {
			out[nextKey] = nextVal
			continue
		}

		prevGroup, prevIsGroup := prevVal.(map[string]any)
		nextGroup, nextIsGroup := nextVal.(map[string]any)
		if !prevIsGroup || !nextIsGroup {
			out[nextKey] = nextVal
			continue
		}

		out[nextKey] = mergeMaps(prevGroup, nextGroup)
	}

	maps.DeleteFunc(out, func(_ string, v any) bool { return v == nil })
	return out
}

func printRecordAttrsJSON(t *testing.T, r slog.Record) {
	t.Helper()

	attrs := st.GetRecordAttrs(r)
	mappedAttrs := mapAttrs(attrs)
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(mappedAttrs)
	t.Logf("%s", buf.String())
}

func requireResultLen[T any](t *testing.T, got []T, expLen int) {
	t.Helper()
	if len(got) != expLen {
		t.Fatalf("wrong number of results; got %d, expected %d", len(got), expLen)
	}
}
