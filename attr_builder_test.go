package slogtesting

import (
	"log/slog"
	"testing"
)

func TestBuildGroupsAroundAttr(t *testing.T) {
	tests := []struct {
		name   string
		groups []string
		attr   slog.Attr
		exp    slog.Attr
	}{
		{
			name:   "empty groups, non-group attr",
			groups: []string{},
			attr:   slog.String("a", "b"),
			exp:    slog.String("a", "b"),
		},
		{
			name:   "groups length 1, non-group attr",
			groups: []string{"G"},
			attr:   slog.String("a", "b"),
			exp:    slog.Attr{Key: "G", Value: slog.GroupValue(slog.String("a", "b"))},
		},
		{
			name:   "groups length 2, non-group attr",
			groups: []string{"G", "H"},
			attr:   slog.String("a", "b"),
			exp: slog.Attr{Key: "G", Value: slog.GroupValue(
				slog.Attr{Key: "H", Value: slog.GroupValue(slog.String("a", "b"))},
			)},
		},
		{
			name:   "empty groups, group attr",
			groups: []string{},
			attr:   slog.Attr{Key: "F", Value: slog.GroupValue(slog.String("a", "b"))},
			exp:    slog.Attr{Key: "F", Value: slog.GroupValue(slog.String("a", "b"))},
		},
		{
			name:   "groups length 1, group attr",
			groups: []string{"G"},
			attr:   slog.Attr{Key: "F", Value: slog.GroupValue(slog.String("a", "b"))},
			exp: slog.Attr{Key: "G", Value: slog.GroupValue(
				slog.Attr{Key: "F", Value: slog.GroupValue(slog.String("a", "b"))},
			)},
		},
		{
			name:   "groups length 2, group attr",
			groups: []string{"G", "H"},
			attr:   slog.Attr{Key: "F", Value: slog.GroupValue(slog.String("a", "b"))},
			exp: slog.Attr{Key: "G", Value: slog.GroupValue(
				slog.Attr{Key: "H", Value: slog.GroupValue(
					slog.Attr{Key: "F", Value: slog.GroupValue(slog.String("a", "b"))},
				)},
			)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := buildGroupsAroundAttr(test.groups, test.attr)
			if !got.Equal(test.exp) {
				t.Errorf(
					"wrong Attr\ngot_key %q, exp_key %q\ngot_val_kind %q, exp_val_kind %q\ngot_val %v exp_val %v",
					got.Key, test.exp.Key, got.Value.Kind().String(), test.exp.Value.Kind().String(), got.Value, test.exp.Value,
				)
			}
		})
	}
}
