//go:build !windows

package lifecycle

import (
	"errors"
)

func discoverCandidatePorts() ([]sessionPort, error) {
	// V3's safe old-session discovery was Windows-specific. V4 preserves that
	// proven boundary instead of pretending to own a cross-platform process scan.
	return []sessionPort{}, nil
}

func stopSessionPID(int) error {
	return errors.New("session retirement is currently implemented for Windows")
}
