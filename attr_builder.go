package slogtesting

import "log/slog"

// attrBuilder is a state mechanism for a call to [slog.Handler.Handle].
type attrBuilder struct {
	replaceAttr func(groups []string, attr slog.Attr) slog.Attr
	attrsByPath map[string]*attrWithPath
	results     []slog.Attr
}

// buildAttr follows the rules stated for [slog.Handler.Handle] that are not
// specific to builtins, and if allowed, constructs an attribute and appends it
// to the results field. As results are updated, so is the attrsByPath field.
// These tandem updates assist in finding, creating and updating attributes
// arranged in groups.
func (ab *attrBuilder) buildAttr(groups []string, attr slog.Attr) {
	// From slog handler docs:
	// 	Attr's values should be resolved.
	// This also happens in some other places.
	attr.Value = attr.Value.Resolve()
	if rep := ab.replaceAttr; rep != nil && attr.Value.Kind() != slog.KindGroup {
		attr = rep(groups, attr)
		// Resolve again in case replaceAttr returns an unresolved attribute.
		attr.Value = attr.Value.Resolve()
	}

	// From slog handler docs:
	// 	If an Attr's key and value are both the zero value, ignore the Attr.
	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		groupAttrs := attr.Value.Group()
		if len(groupAttrs) < 1 {
			// From slog handler docs:
			// 	If a group has no Attrs (even if it has a non-empty key), ignore it.
			return
		}

		if attr.Key != "" {
			// Ensure group attributes are properly placed.
			groups = append(groups, attr.Key)
		}

		for i, a := range groupAttrs {
			ab.buildAttr(groups, a)
			// Resolve each group attribute. Though this attribute was passed to
			// buildAttr, which calls Resolve, its resolution wouldn't be observed
			// here because it's passed as a value.
			groupAttrs[i].Value = a.Value.Resolve()
		}

		if len(groups) > 0 {
			return
		}

		attr = slog.GroupAttrs(attr.Key, groupAttrs...)
		ab.results = append(ab.results, attr)
		ab.attrsByPath[attr.Key] = newAttrWithPath(&attr)
		return
	}

	if len(groups) < 1 {
		ab.results = append(ab.results, attr)
		ab.attrsByPath[attr.Key] = newAttrWithPath(&attr)
		return
	}

	mount, path := findAttr(ab.attrsByPath, groups)
	slog.Debug(logPrefix+"from (*ab).buildAttr, after findAttr",
		slog.Any("input_groups", groups), slog.String("input_attr_key", attr.Key), slog.Any("input_attr_value", attr.Value),
		slog.Bool("is_mount_nil", mount == nil), slog.Any("path", path),
	)

	if mount == nil {
		attr = buildGroupsAroundAttr(groups, attr)
		ab.results = append(ab.results, attr)

		// Ensure that the item to map is the same item added to results. In
		// other spots where the results field is updated in tandem with the
		// attrsByPath map, it's enough to pass a pointer to the local variable,
		// attr. For this case, obtain a pointer to the last item in the struct
		// field, results, and add that to the map.
		//
		// As of 2025-11 it's unclear why, but the tests do not pass if a
		// pointer to a local variable is used in the same way as the other
		// updates to the results and attrsByPath fields. This may indicate a
		// subtle bug, and is something to watch out for.
		lastItem := &ab.results[len(ab.results)-1]
		ab.attrsByPath[attr.Key] = newAttrWithPath(lastItem)
		return
	}

	if len(path) < len(groups) {
		// The findAttr function resulted in a partial match. Build the remaining path.
		remainingPath := groups[len(path):]
		attr = buildGroupsAroundAttr(remainingPath, attr)
	}

	if mnt := mount.Attr; mnt.Value.Kind() == slog.KindGroup {
		groupAttrs := append(mnt.Value.Group(), attr)
		mnt.Value = slog.GroupValue(groupAttrs...)
		mount.Attr = mnt
		mount.children[attr.Key] = newAttrWithPath(&attr)
	} else {
		slog.Debug(logPrefix+"from (*ab).buildAttr, would mount a non-group attr onto a non-group attr",
			slog.GroupAttrs("mount", slog.String("key", mount.Key), slog.String("val_kind", mount.Value.Kind().String())),
			slog.GroupAttrs("attr", slog.String("key", attr.Key), slog.String("val_kind", attr.Value.Kind().String())),
		)
	}
}

func buildGroupsAroundAttr(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) < 1 {
		return attr
	}

	// Start at the end of groups and build the output backwards. This results
	// in the first group being the primary group for attr.
	out := slog.GroupAttrs(groups[len(groups)-1], attr)

	for i := len(groups) - 2; i >= 0; i-- {
		group := groups[i]
		out = slog.GroupAttrs(group, out)
	}

	return out
}
