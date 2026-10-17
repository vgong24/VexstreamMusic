//go:build !windows && !darwin

// Package picker owns the native desktop folder-selection adapter.
package picker

import "errors"

func Choose() (string, error) {
	return "", errors.New("folder picker unavailable")
}
