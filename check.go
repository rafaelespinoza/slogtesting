package slogtesting

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

// A Check is a general-purpose test on slog attributes. It's based off the
// implementation of the [testing/slogtest] package.
type Check func(attrs []slog.Attr) error

// HasKey makes a Check for the presence of an attribute with a key.
// The Check will return an error unless a matching attribute is found in attrs.
// Use this when you want to target an attribute with that key but do not care
// about the value.
func HasKey(key string) Check {
	return func(attrs []slog.Attr) (err error) {
		matchKey := makeKeyMatcher(key)
		got := collectMatchingAttrs(attrs, matchKey)
		if len(got) < 1 {
			err = fmt.Errorf("did not find expected key %s", key)
		}
		return
	}
}

// MissingKey makes a Check for the absence of an attribute with a key.
// The Check will return an error if a matching attribute is found in attrs.
func MissingKey(key string) Check {
	return func(attrs []slog.Attr) (err error) {
		matchKey := makeKeyMatcher(key)
		got := collectMatchingAttrs(attrs, matchKey)
		if len(got) > 0 {
			err = fmt.Errorf("unexpected key %s", key)
		}
		return
	}
}

// HasAttr makes a Check for the presence of an attribute with the wanted
// key and value.
// The Check will return an error unless a matching attribute is found in attrs.
// Use this when you want to see exactly 1 attribute with an equal key and value
// in attrs.
func HasAttr(want slog.Attr) Check {
	return func(attrs []slog.Attr) (err error) {
		matchKey := makeKeyMatcher(want.Key)
		gotMatches, err := collectNMatchingAttrs(attrs, 1, matchKey)
		if err != nil {
			err = fmt.Errorf("looking for attr with key %s: %v", want.Key, err)
			return
		}

		got := gotMatches[0]
		if !got.Equal(want) {
			err = fmt.Errorf(
				"attributes not equal\ngot_key %q, want_key %q\ngot_val_kind %q, want_val_kind %q\ngot_val %v want_val %v",
				got.Key, want.Key, got.Value.Kind().String(), want.Value.Kind().String(), got.Value, want.Value,
			)
		}
		return
	}
}

// HasMatch makes a Check that allows stricter or looser attribute targeting
// logic than what's provided by the other check functions.
// The Check will return an error unless a matching attribute is found in attrs.
func HasMatch(m func(slog.Attr) bool) Check {
	return func(attrs []slog.Attr) (err error) {
		_, err = collectNMatchingAttrs(attrs, 1, m)
		return
	}
}

// InGroup makes a Check for a Check in a group with a matching name. The output
// Check will first look for group with a given name, then run all of the input
// Checks upon the attributes in the group, and then combine non-nil errors into
// 1 using [errors.Join].
func InGroup(name string, c Check, moreChecks ...Check) Check {
	return func(attrs []slog.Attr) error {
		matchKey := makeKeyMatcher(name)
		got, err := collectNMatchingAttrs(attrs, 1, matchKey)
		if err != nil {
			// Though there is only 1 error here, keep the interface consistent
			// with cases where multiple errors are combined using errors.Join.
			// The wanted effect is that the output error implements the method
			// `Unwrap() []error`.
			err = fmt.Errorf("looking for group attr with name %s: %v", name, err)
			return errors.Join(err)
		}

		kind := got[0].Value.Kind()
		if kind != slog.KindGroup {
			// Same idea as noted above. There's only 1 error here, but keep the
			// interface consistent. Ensure that the output error implements the
			// method `Unwrap() []error`.
			err = fmt.Errorf("wrong kind (%s) for item with key %s, expected %s", kind, name, slog.KindGroup.String())
			return errors.Join(err)
		}

		errs := make([]error, 0, 1+len(moreChecks))
		groupVals := got[0].Value.Group()
		if err := c(groupVals); err != nil {
			err = makeErrorWithGroupPath(err, name)
			errs = append(errs, err)
		}
		for _, check := range moreChecks {
			if err := check(groupVals); err != nil {
				err = makeErrorWithGroupPath(err, name)
				errs = append(errs, err)
			}
		}
		errs = slices.Clip(errs)
		return errors.Join(errs...)
	}
}

func makeErrorWithGroupPath(err error, groupName string) error {
	var errWithPath *errorWithGroupPath
	if !errors.As(err, &errWithPath) {
		err = &errorWithGroupPath{
			err:       err,
			groupPath: []string{groupName},
		}
	} else {
		// Add the new value at the start of the slice b/c the targeted error
		// occurred "deeper" within the attribute groups, and we want for the
		// presentation of group names to be from the outermost to innermost
		// group names.
		errWithPath.groupPath = append([]string{groupName}, errWithPath.groupPath...)
		err = errWithPath
	}
	return err
}

type errorWithGroupPath struct {
	err       error
	groupPath []string
}

func (e *errorWithGroupPath) Error() string {
	return fmt.Sprintf("%v; group path %s", e.err, strings.Join(e.groupPath, "."))
}

func (e *errorWithGroupPath) GroupPath() []string { return e.groupPath }
