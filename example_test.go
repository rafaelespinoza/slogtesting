package slogtesting_test

import (
	"fmt"
	"log/slog"

	st "github.com/rafaelespinoza/slogtesting"
)

func Example() {
	// Each record output by the logger is captured here.
	var records []slog.Record
	capture := func(r slog.Record) error { records = append(records, r); return nil }

	// Create handler and logger.
	opts := st.AttrHandlerOptions{CaptureRecord: capture}
	handler := st.NewAttrHandler(&opts) // the default level is INFO.
	logger := slog.New(handler)

	// Your application does something that needs a test. This example
	// accumulates some data and outputs a record at the INFO level.
	//
	// This output invocation will be recorded b/c the handler's logging
	// level will allow calls at the INFO level.
	// If the handler was a *slog.TextHandler, the output would look similar to:
	// 	time=2006-01-02T15:04:05.012Z level=INFO msg=msg a=b G.c=d G.H.e=f
	logger.With("a", "b").WithGroup("G").With("c", "d").WithGroup("H").Info("msg", "e", "f")
	// This output won't be recorded b/c the level of the underlying handler
	// is above DEBUG.
	logger.Debug("You won't see me")

	if len(records) != 1 {
		fmt.Printf("wrong number of captured records; got %d, expected %d", len(records), 1)
		return
	}

	// This is the output data to test. Collect the attributes for each record
	// that was output by the logger. It will also include the built-in
	// attributes: time, level, message.
	attrs := st.GetRecordAttrs(records[0])

	// Run these tests.
	checks := []struct {
		check st.Check
		okMsg string
	}{
		{
			check: st.HasKey(slog.TimeKey),
			okMsg: "found key " + slog.TimeKey,
		},
		{
			check: st.HasKey(slog.LevelKey),
			okMsg: "found key " + slog.LevelKey,
		},
		{
			check: st.HasAttr(slog.String(slog.MessageKey, "msg")),
			okMsg: "found attribute with key " + slog.MessageKey,
		},
		{
			check: st.HasAttr(slog.String("a", "b")),
			okMsg: "found attribute with key a",
		},
		{
			check: st.InGroup("G", st.HasAttr(slog.String("c", "d"))),
			okMsg: "found group G and attribute with key c",
		},
		{
			check: st.InGroup("G", st.InGroup("H", st.HasAttr(slog.String("e", "f")))),
			okMsg: "found group G, another group H and attribute with key c",
		},
		{
			check: st.MissingKey("z"),
			okMsg: "did not find attribute with key z",
		},
	}

	for _, ex := range checks {
		err := ex.check(attrs)
		if err != nil {
			fmt.Printf("unexpected error %v\n", err)
		} else {
			fmt.Println(ex.okMsg)
		}
	}
	// Output:
	// found key time
	// found key level
	// found attribute with key msg
	// found attribute with key a
	// found group G and attribute with key c
	// found group G, another group H and attribute with key c
	// did not find attribute with key z
}
