//go:build !windows

package youtube

import "testing"

func TestProviderSubprocessDoesNotInventPlatformAttributes(t *testing.T) {
	if attr := hiddenProcAttr(); attr != nil {
		t.Fatalf("unexpected non-Windows provider process attributes: %+v", attr)
	}
}
