package slogtesting

import (
	"log/slog"
	"slices"
	"testing"
)

func pointTo[T any](in T) (out *T) { return &in }

func TestFindAttr(t *testing.T) {
	builtinNode := attrWithPath{Attr: pointTo(slog.String(slog.MessageKey, "message")), children: nil}
	terminalNode := attrWithPath{Attr: pointTo(slog.String("c", "charlie")), children: nil}

	nodeH := attrWithPath{
		Attr:     pointTo(slog.String("H", "Hotel")),
		children: nil,
	}

	nodeG := attrWithPath{
		Attr: pointTo(slog.GroupAttrs("G", *terminalNode.Attr, *nodeH.Attr)),
		children: map[string]*attrWithPath{
			"child": &terminalNode,
			"H":     &nodeH,
		},
	}

	tree := map[string]*attrWithPath{
		slog.MessageKey: &builtinNode,
		"G":             &nodeG,
	}

	msgNode := tree[slog.MessageKey]

	tests := []struct {
		name    string
		tree    map[string]*attrWithPath
		path    []string
		expAttr *slog.Attr
		expPath []string
	}{
		{
			name:    "top-level found",
			tree:    tree,
			path:    []string{slog.MessageKey},
			expAttr: msgNode.Attr,
			expPath: []string{slog.MessageKey},
		},
		{
			name:    "top-level not found",
			tree:    tree,
			path:    []string{"foo"},
			expAttr: nil,
		},
		{
			name:    "sub-level full match found partway",
			tree:    tree,
			path:    []string{"G"},
			expAttr: nodeG.Attr,
			expPath: []string{"G"},
		},
		{
			name:    "sub-level no match",
			tree:    tree,
			path:    []string{"G", "sibling"},
			expAttr: nodeG.Attr,
			expPath: []string{"G"},
		},
		{
			name:    "sub-level full match found deep",
			tree:    tree,
			path:    []string{"G", "H"},
			expAttr: nodeH.Attr,
			expPath: []string{"G", "H"},
		},
		{
			name:    "sub-level partial match",
			tree:    tree,
			path:    []string{"G", "notfound"},
			expAttr: nodeG.Attr,
			expPath: []string{"G"},
		},
		{
			name:    "sub-level path too long",
			tree:    tree,
			path:    []string{"G", "H", "I"},
			expAttr: nodeH.Attr,
			expPath: []string{"G", "H"},
		},
		{
			name:    "reaches too deep from top level",
			tree:    tree,
			path:    []string{slog.MessageKey, "x"},
			expAttr: msgNode.Attr,
			expPath: []string{slog.MessageKey},
		},
		{
			name:    "empty",
			tree:    tree,
			path:    []string{},
			expAttr: nil,
			expPath: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotNode, gotPath := findAttr(test.tree, test.path)

			if !slices.Equal(gotPath, test.expPath) {
				t.Errorf("wrong path; got %q, expected %q", gotPath, test.expPath)
			}

			if gotNode == nil && test.expAttr != nil {
				t.Error("got nil node, but expected non-nil node")
				return
			} else if gotNode != nil && test.expAttr == nil {
				t.Error("got non-nil node, but expected nil node")
				return
			} else if gotNode == nil && test.expAttr == nil {
				return // OK
			}

			if !gotNode.Equal(*test.expAttr) {
				t.Errorf(
					"wrong node Attr\ngot_key %q, exp_key %q\ngot_val_kind %q, exp_val_kind %q\ngot_val %v exp_val %v",
					gotNode.Key, test.expAttr.Key, gotNode.Value.Kind().String(), test.expAttr.Value.Kind().String(), gotNode.Value, test.expAttr.Value,
				)
			}
		})
	}
}
