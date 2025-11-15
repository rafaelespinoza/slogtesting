package slogtesting

import (
	"log/slog"
	"slices"
	"testing"
)

func TestApplyGroupsOrAttrs(t *testing.T) {
	type CallbackArgs struct {
		Groups []string
		Attr   slog.Attr
	}

	tests := []struct {
		name            string
		setup           func() *groupsOrAttrs
		expCalls        []CallbackArgs
		expOutputGroups []string
	}{
		{
			name:  "empty",
			setup: func() *groupsOrAttrs { return nil },
		},
		{
			name: "1 group",
			setup: func() *groupsOrAttrs {
				return &groupsOrAttrs{Group: "a"}
			},
			expCalls:        []CallbackArgs{},
			expOutputGroups: []string{"a"},
		},
		{
			name: "1 group empty",
			setup: func() *groupsOrAttrs {
				return &groupsOrAttrs{Group: ""}
			},
			expCalls:        []CallbackArgs{},
			expOutputGroups: []string{},
		},
		{
			name: "1 attr",
			setup: func() *groupsOrAttrs {
				return &groupsOrAttrs{Attrs: []slog.Attr{slog.String("a", "aaa")}}
			},
			expCalls: []CallbackArgs{
				{Attr: slog.String("a", "aaa")},
			},
		},
		{
			name: "next - only groups",
			setup: func() *groupsOrAttrs {
				next := groupsOrAttrs{Group: "a"}
				return &groupsOrAttrs{Group: "b", Next: &next}
			},
			expCalls:        []CallbackArgs{},
			expOutputGroups: []string{"a", "b"},
		},
		{
			name: "next - only attrs",
			setup: func() *groupsOrAttrs {
				next := groupsOrAttrs{Attrs: []slog.Attr{slog.String("a", "aaa")}}
				return &groupsOrAttrs{Attrs: []slog.Attr{slog.String("b", "bbb")}, Next: &next}
			},
			expCalls: []CallbackArgs{
				{Attr: slog.String("a", "aaa")},
				{Attr: slog.String("b", "bbb")},
			},
		},
		{
			name: "next - groups and attrs",
			setup: func() *groupsOrAttrs {
				next := groupsOrAttrs{Group: "G"}
				return &groupsOrAttrs{Attrs: []slog.Attr{slog.String("a", "aaa")}, Next: &next}
			},
			expCalls: []CallbackArgs{
				{Groups: []string{"G"}, Attr: slog.String("a", "aaa")},
			},
			expOutputGroups: []string{"G"},
		},
		{
			name: "next - attrs and groups",
			setup: func() *groupsOrAttrs {
				next := groupsOrAttrs{Attrs: []slog.Attr{slog.String("a", "aaa")}}
				return &groupsOrAttrs{Group: "G", Next: &next}
			},
			expCalls: []CallbackArgs{
				{Attr: slog.String("a", "aaa")},
			},
			expOutputGroups: []string{"G"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			receivedArgs := []CallbackArgs{}
			got := applyGroupsOrAttrs(test.setup(), func(groups []string, a slog.Attr) {
				receivedArgs = append(receivedArgs, CallbackArgs{Groups: groups, Attr: a})
			})

			if !slices.Equal(got, test.expOutputGroups) {
				t.Errorf("unexpected output\ngot: %q\nexp: %q", got, test.expOutputGroups)
			}

			if len(receivedArgs) != len(test.expCalls) {
				t.Fatalf("wrong number of invocations; got %d, expected %d", len(receivedArgs), len(test.expCalls))
			}

			for i, arg := range receivedArgs {
				exp := test.expCalls[i]

				if !slices.Equal(arg.Groups, exp.Groups) {
					t.Errorf("unexpected groups at invocation %d\ngot: %q\nexp: %q", i, arg.Groups, exp.Groups)
				}
				if !arg.Attr.Equal(exp.Attr) {
					t.Errorf("unexpected attr at invocation %d\ngot: %q\nexp: %q", i, arg.Attr, exp.Attr)
				}
			}
		})
	}
}
