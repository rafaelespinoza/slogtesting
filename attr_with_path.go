package slogtesting

import "log/slog"

// attrWithPath assists in mapping attributes within handler state to help
// finding attributes arranged in a hierarchical structure. In slog terms, these
// are groups.
type attrWithPath struct {
	*slog.Attr
	children map[string]*attrWithPath
}

func newAttrWithPath(attr *slog.Attr) *attrWithPath {
	return &attrWithPath{
		Attr:     attr,
		children: make(map[string]*attrWithPath),
	}
}

// findAttr searches through root for an attribute along inPath. It returns a
// node and the path taken to get there. A node targeted by inPath may not be
// found in root, in which case an ancestor of a hypothetical node is returned.
// Use this function to help find a place in root to insert a new attribute.
// There are a few outcomes for this function:
//   - The input path is an exact match for an existing item. It returns the
//     found node and the path it took to get there. This path would be identical
//     to the input path.
//   - The input path is a partial match for an existing item. Though it didn't
//     find an existing attribute with the exact path, it returns the closest
//     existing node and returns the path it took to get there. This path would
//     be the first n steps of inPath. You may want to use the remainder of the
//     path to create steps (groups) to an destination node in this case.
//   - The input path corresponds to no nodes in the tree. The return values are
//     both empty.
//   - As a special case, if the inPath is length 0, then it returns nil, nil.
//     To the caller, this might mean to create an attribute at the top level.
func findAttr(root map[string]*attrWithPath, inPath []string) (*attrWithPath, []string) {
	if len(inPath) < 1 {
		return nil, nil
	}

	curr := root
	var found *attrWithPath
	// pathInProgress records the path taken so far. In case a node was not
	// found, at least we know how far we got; and compared to the input path,
	// we know which steps to take to build to that target node.
	pathInProgress := []string{}

	for i, key := range inPath {
		node, ok := curr[key]
		if !ok {
			// Return the last successfully found node and its path.
			if found == nil {
				slog.Debug(
					logPrefix+"from findAttr, key not found",
					slog.String("key", key), slog.Int("depth", i),
				)
				return nil, pathInProgress
			}

			slog.Debug(
				logPrefix+"from findAttr, key not found; returning parent",
				slog.String("key", key), slog.Int("depth", i), slog.String("parent_key", found.Attr.Key),
			)
			return found, pathInProgress
		}

		found = node
		pathInProgress = append(pathInProgress, key)

		// If this is the last element of the path, an exact match is found.
		if i == len(inPath)-1 {
			return found, pathInProgress
		}

		// Prepare for the next iteration: Check for children. If the node has
		// no children but the input path continues, then this probably means
		// the caller would need to create at least 1 level of groups to reach
		// the end of the input path.
		if found.children == nil {
			slog.Debug(
				logPrefix+"from findAttr, traversal stopped; expected children for remaining path, but found none",
				slog.String("stopped_at_key", found.Attr.Key),
			)

			// The path was successfully matched up to this point.
			return found, pathInProgress
		}

		curr = found.children
	}

	return nil, nil
}
