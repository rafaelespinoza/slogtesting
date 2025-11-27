package slogtesting_test

import (
	"log/slog"
	"slices"
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

func TestCheckGroupsFailures(t *testing.T) {
	tests := []struct {
		name          string
		attrs         []slog.Attr
		check         st.Check
		expGroupPaths [][]string // 1 slice per expected errorWithGroupPath
	}{
		{
			name: "group attr not found",
			attrs: []slog.Attr{
				slog.String("c", "d"),
			},
			check:         st.InGroup("G", st.HasKey("c")),
			expGroupPaths: [][]string{},
		},
		{
			name: "nested group attr not found",
			attrs: []slog.Attr{
				slog.GroupAttrs("G",
					slog.String("c", "d"),
				),
			},
			check:         st.InGroup("G", st.InGroup("H", st.HasKey("d"))),
			expGroupPaths: [][]string{{"G"}},
		},
		{
			name: "attr found but is not a group",
			attrs: []slog.Attr{
				slog.GroupAttrs("G",
					slog.String("c", "d"),
				),
			},
			check:         st.InGroup("G", st.InGroup("c", st.HasKey("c"))),
			expGroupPaths: [][]string{{"G"}},
		},
		{
			name: "1 level deep",
			attrs: []slog.Attr{
				slog.GroupAttrs("G",
					slog.String("c", "d"),
				),
			},
			check:         st.InGroup("G", st.HasKey("d")),
			expGroupPaths: [][]string{{"G"}},
		},
		{
			name: "2 levels deep",
			attrs: []slog.Attr{
				slog.GroupAttrs("G",
					slog.GroupAttrs("H",
						slog.String("e", "f"),
					),
				),
			},
			check:         st.InGroup("G", st.InGroup("H", st.HasKey("f"))),
			expGroupPaths: [][]string{{"G", "H"}},
		},
		{
			name: "multiple errors different groups",
			attrs: []slog.Attr{
				slog.GroupAttrs("G",
					slog.GroupAttrs("H",
						slog.String("e", "f"),
					),
					slog.String("a", "b"),
				),
			},
			check: st.InGroup("G",
				st.InGroup("H", st.HasKey("f")),
				st.HasKey("b"),
			),
			expGroupPaths: [][]string{{"G", "H"}, {"G"}},
		},
	}

	type (
		// An unwrappableErrors value is returned by errors.Join.
		unwrappableErrors interface{ Unwrap() []error }
		// An errorWithGroupPath has info about the path to a group for an error.
		errorWithGroupPath interface{ GroupPath() []string }
	)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.check(test.attrs)
			if err == nil {
				t.Fatal("expected non-empty error")
			}
			t.Log(err)

			unwrappableErr, ok := err.(unwrappableErrors)
			if !ok {
				// The documentation for this function is that the output error
				// would be treated with errors.Join. An effect of using
				// errors.Join is that output error also implements this
				// interface:  `Unwrap() []error`
				t.Fatal("expected error to implement expected interface with Unwrap method")
			}

			if len(test.expGroupPaths) < 1 {
				return
			}

			unwrappedErrs := unwrappableErr.Unwrap()
			if len(unwrappedErrs) != len(test.expGroupPaths) {
				t.Fatalf("wrong number of errors; got %d, expected %d", len(unwrappedErrs), len(test.expGroupPaths))
			}
			for i, uerr := range unwrappedErrs {
				errWithPath, ok := uerr.(errorWithGroupPath)
				if !ok {
					t.Fatal("expected error to implement expected interface with GroupPath method")
				}

				gotGroupPath := errWithPath.GroupPath()
				expErrKeys := test.expGroupPaths[i]
				if !slices.Equal(gotGroupPath, expErrKeys) {
					t.Errorf("group path wrong; got %q, expected %q", gotGroupPath, expErrKeys)
				}
			}
		})
	}
}
