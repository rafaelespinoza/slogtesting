package slogtesting_test

import (
	"log/slog"
	"testing"
	"time"

	st "github.com/rafaelespinoza/slogtesting"
)

func TestCheck(t *testing.T) {
	attrs := []slog.Attr{slog.String("foo", "bar")}
	checks := []st.Check{
		st.HasKey("foo"),
		st.MissingKey("bar"),
		st.HasAttr(slog.String("foo", "bar")),
	}
	for _, check := range checks {
		err := check(attrs)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestCheckGroups(t *testing.T) {
	attrs := []slog.Attr{
		slog.Time(slog.TimeKey, time.Now()),
		slog.String(slog.LevelKey, slog.LevelInfo.String()),
		slog.String(slog.MessageKey, "msg"),
		slog.String("a", "b"),
		{
			Key: "G",
			Value: slog.GroupValue(
				slog.String("c", "d"),
				slog.Attr{
					Key:   "H",
					Value: slog.GroupValue(slog.String("e", "f")),
				},
			),
		},
	}

	tests := []struct {
		name  string
		check st.Check
	}{
		{
			name:  "has key time",
			check: st.HasKey(slog.TimeKey),
		},
		{
			name:  "has key level",
			check: st.HasAttr(slog.String(slog.LevelKey, slog.LevelInfo.String())),
		},
		{
			name:  "has attr msg",
			check: st.HasAttr(slog.String(slog.MessageKey, "msg")),
		},
		{
			name: "in group g",
			check: st.InGroup("G",
				st.HasAttr(slog.String("c", "d")),
				st.InGroup("H",
					st.HasAttr(slog.String("e", "f")),
					st.MissingKey("y"),
				),
				st.MissingKey("z"),
			),
		},
		{
			name:  "missing key d",
			check: st.MissingKey("d"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.check(attrs)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestCheckFailures(t *testing.T) {
	attrs := []slog.Attr{
		slog.String("bar", "foo"),
		slog.String("g", "golf"),

		slog.String("duplicate_key", "d"),
		slog.String("duplicate_key", "d"),

		{Key: "dupe_group", Value: slog.GroupValue(slog.String("d", "g"))},
		{Key: "dupe_group", Value: slog.GroupValue(slog.String("d", "g"))},

		{Key: "group_with_dupes", Value: slog.GroupValue(slog.String("d", "g"), slog.String("d", "g"))},
	}

	tests := []struct {
		name  string
		check st.Check
	}{
		{
			name:  "has key",
			check: st.HasKey("foo"),
		},
		{
			name:  "missing key",
			check: st.MissingKey("bar"),
		},
		{
			name:  "has attr - key wrong",
			check: st.HasAttr(slog.String("foo", "bar")),
		},
		{
			name:  "has attr - val wrong",
			check: st.HasAttr(slog.String("bar", "food")),
		},
		{
			name:  "has attr - enforces 1 attribute with key",
			check: st.HasAttr(slog.String("duplicate_key", "d")),
		},
		{
			name:  "group - attr not found",
			check: st.InGroup("H", st.HasKey("h")),
		},
		{
			name:  "group - attr found, not a group",
			check: st.InGroup("g", st.HasKey("golf")),
		},
		{
			name:  "group - enforces 1 attribute with key",
			check: st.InGroup("dupe_group", st.HasKey("d")),
		},
		{
			name:  "group - enforces 1 attribute with key in a group",
			check: st.InGroup("group_with_dupes", st.HasAttr(slog.String("d", "g"))),
		},
		{
			name:  "group - detects error when check is a variadic arg",
			check: st.InGroup("group_with_dupes", st.MissingKey("z"), st.HasAttr(slog.String("d", "g"))),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.check(attrs)
			if err == nil {
				t.Fatal("expected an error but got nil")
			}
			t.Log(err)
		})
	}
}
