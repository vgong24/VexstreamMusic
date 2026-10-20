//go:build !windows

package youtube

import (
	"context"
	"testing"
)

func TestProviderSubprocessDoesNotInventPlatformAttributes(t *testing.T) {
	cmd := providerCommand(
		context.Background(),
		[]string{"sh", "-c", "true"},
	)
	if attr := cmd.SysProcAttr; attr != nil {
		t.Fatalf("unexpected non-Windows provider process attributes: %+v", attr)
	}
}
