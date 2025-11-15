package slogtesting

import "log/slog"

// groupsOrAttrs holds either a group name or a list of slog.Attrs. It's a
// simple linked list for helping slog.Handler implementations accumulate data.
// This type is lifted from [github.com/jba/slog/withsupport].
type groupsOrAttrs struct {
	Group string      // group name if non-empty
	Attrs []slog.Attr // attrs if non-empty
	Next  *groupsOrAttrs
}

// applyGroupsOrAttrs calls f on each Attr in g. The first argument to f is the
// list of groups that precede the Attr. Apply returns the complete list of
// groups. Use this in a handler's Handle method to access the handler's
// accumulated data.
func applyGroupsOrAttrs(g *groupsOrAttrs, f func(groups []string, a slog.Attr)) []string {
	var groups []string

	var rec func(*groupsOrAttrs)
	rec = func(g *groupsOrAttrs) {
		if g == nil {
			return
		}
		rec(g.Next)
		if g.Group != "" {
			groups = append(groups, g.Group)
		} else {
			for _, a := range g.Attrs {
				f(groups, a)
			}
		}
	}
	rec(g)

	return groups
}
