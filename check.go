package slogtesting

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
)

// A Check is a general-purpose test on slog attributes. It's based off the
// implementation of the [testing/slogtest] package.
type Check func([]slog.Attr) error

// HasKey makes a Check for the presence of an attribute with a key.
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

// InGroup makes a Check for a Check in a group with a matching name. The
// output Check will run all of the input Checks and combine non-nil errors into
// 1 using errors.Join.
func InGroup(name string, c Check, moreChecks ...Check) Check {
	return func(attrs []slog.Attr) (err error) {
		matchKey := makeKeyMatcher(name)
		got, err := collectNMatchingAttrs(attrs, 1, matchKey)
		if err != nil {
			err = fmt.Errorf("looking for group attr with name %s: %v", name, err)
			return
		}

		kind := got[0].Value.Kind()
		if kind != slog.KindGroup {
			err = fmt.Errorf("wrong kind (%s) for item with key %s, expected %s", kind, name, slog.KindGroup.String())
			return
		}

		errs := make([]error, 0, 1+len(moreChecks))
		groupVals := got[0].Value.Group()
		if err = c(groupVals); err != nil {
			errs = append(errs, err)
		}
		for _, check := range moreChecks {
			if err = check(groupVals); err != nil {
				errs = append(errs, err)
			}
		}
		errs = slices.Clip(errs)
		return errors.Join(errs...)
	}
}
