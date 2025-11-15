package slogtesting_test

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	st "github.com/rafaelespinoza/slogtesting"
)

func TestCaptureRecords(t *testing.T) {
	testErr := errors.New("test")

	// newRecordWithAttrs mimics the slog.Record construction behavior of this
	// package handler so that expected test ouputs are in the ballpark of what
	// the handler actually produces.
	newRecordWithAttrs := func(lvl slog.Level, msg string, as ...slog.Attr) slog.Record {
		out := slog.NewRecord(time.Now(), lvl, msg, 1)
		// Add record fields to the record's internal attributes to match the
		// behavior of this package's slog.Handler.
		attrs := []slog.Attr{
			slog.Time(slog.TimeKey, out.Time),
			slog.String(slog.LevelKey, out.Level.String()),
			slog.String(slog.MessageKey, out.Message),
		}
		out.AddAttrs(append(attrs, as...)...)
		return out
	}

	// numCaptRecCalls tracks uses of an input CaptureRecord function. It's
	// reset for every test case.
	var numCaptRecCalls int

	tests := []struct {
		name            string
		opts            *st.AttrHandlerOptions
		run             func(slog.Handler) error
		expCaptRecCalls int
		expOut          []slog.Record
		expErr          error
	}{
		{
			name: "ok",
			opts: &st.AttrHandlerOptions{},
			run: func(h slog.Handler) error {
				lgr := slog.New(h)
				lgr.Info("msg")
				return nil
			},
			expOut: []slog.Record{newRecordWithAttrs(slog.LevelInfo, "msg")},
		},
		{
			name: "ok - empty options",
			run: func(h slog.Handler) error {
				lgr := slog.New(h)
				lgr.Info("msg")
				return nil
			},
			expOut: []slog.Record{newRecordWithAttrs(slog.LevelInfo, "msg")},
		},
		{
			name: "ok - handler level warn",
			opts: &st.AttrHandlerOptions{HandlerOptions: slog.HandlerOptions{Level: slog.LevelWarn}},
			run: func(h slog.Handler) error {
				lgr := slog.New(h)
				lgr.Info("information")
				lgr.Warn("a little awkward but ok", slog.Any("error", testErr))
				return nil
			},
			expOut: []slog.Record{
				newRecordWithAttrs(slog.LevelWarn, "a little awkward but ok", slog.Any("error", testErr)),
			},
		},
		{
			name: "test returns error",
			opts: &st.AttrHandlerOptions{},
			run: func(h slog.Handler) error {
				lgr := slog.New(h)
				lgr.Info("msg")
				return fmt.Errorf("%w", testErr)
			},
			expOut: []slog.Record{newRecordWithAttrs(slog.LevelInfo, "msg")},
			expErr: testErr,
		},
		{
			name: "ok - capture record func",
			opts: &st.AttrHandlerOptions{
				CaptureRecord: func(r slog.Record) error {
					numCaptRecCalls++
					return nil
				},
			},
			run: func(h slog.Handler) error {
				lgr := slog.New(h)
				lgr.Info("msg")
				return nil
			},
			expCaptRecCalls: 1,
			expOut: []slog.Record{
				newRecordWithAttrs(slog.LevelInfo, "msg"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(func() { numCaptRecCalls = 0 })
			records, err := st.CaptureRecords(test.opts, test.run)

			if numCaptRecCalls != test.expCaptRecCalls {
				t.Errorf(
					"unexpected number of calls to input CaptureRecord func; got %d, expected %d",
					numCaptRecCalls, test.expCaptRecCalls,
				)
			}

			if test.expErr != nil && err == nil {
				t.Fatalf("expected an error (%v) but got nil", test.expErr)
			} else if test.expErr == nil && err != nil {
				t.Fatalf("unexpected error %v", err)
			} else if test.expErr != nil && err != nil && !errors.Is(err, test.expErr) {
				t.Fatalf("wrong error; got %v, expected %v", err, test.expErr)
			}

			if len(records) != len(test.expOut) {
				t.Fatalf("wrong number of output records; got %d, expected %d", len(records), len(test.expOut))
			}

			for i, gotRecord := range records {
				expRecord := test.expOut[i]

				if gotRecord.Time.IsZero() {
					t.Errorf("record[%d] expected non-zero Time", i)
				}

				if gotRecord.Message != expRecord.Message {
					t.Errorf(
						"record[%d] wrong Message; got %q, expected %q",
						i, gotRecord.Message, expRecord.Message,
					)
				}

				if gotRecord.Level != expRecord.Level {
					t.Errorf(
						"record[%d] wrong Level; got %q, expected %q",
						i, gotRecord.Level.String(), expRecord.Level.String(),
					)
				}

				if gotRecord.PC == 0 {
					t.Errorf("record[%d] expected non-empty PC", i)
				}

				if gotRecord.NumAttrs() != expRecord.NumAttrs() {
					t.Fatalf(
						"record[%d] wrong number of attrs; got %d, expected %d",
						i, gotRecord.NumAttrs(), expRecord.NumAttrs(),
					)
				}

				expAttrs := st.GetRecordAttrs(expRecord)

				for j, gotAttr := range st.GetRecordAttrs(gotRecord) {
					expAttr := expAttrs[j]

					if gotAttr.Key == slog.TimeKey {
						// Testing time can be tricky because it is a moving
						// target. Some other logging tests will force a
						// consistent time in tests by using the ReplaceAttrs
						// function. But we cannot do that with the API under
						// test b/c there are cases where we want for the input
						// options to be nil. To accommodate that, loosen the
						// expectations for the value.
						if gotAttr.Value.Time().IsZero() {
							t.Errorf("record[%d], Value for Attr with key %s should be non-zero", i, gotAttr.Key)
						}
						continue
					}

					if !gotAttr.Equal(expAttr) {
						t.Errorf(
							"record[%d] wrong Attr at position [%d]\ngot_key %q, exp_key %q\ngot_val_kind %q, exp_val_kind %q\ngot_val %v exp_val %v",
							i, j, gotAttr.Key, expAttr.Key, gotAttr.Value.Kind().String(), expAttr.Value.Kind().String(), gotAttr.Value, expAttr.Value,
						)
					}
				}
			}
		})
	}
}
