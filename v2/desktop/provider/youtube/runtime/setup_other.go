//go:build !windows && !darwin

package runtimeenv

import (
	"context"
	"errors"
)

func Setup(context.Context, string) (string, error) {
	return "", errors.New("automatic YouTube runtime setup is not implemented on this platform")
}
