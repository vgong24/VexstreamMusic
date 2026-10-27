//go:build windows

package youtube

import (
	"context"
	"testing"
)

func TestProviderSubprocessStaysHiddenOnWindows(t *testing.T) {
	cmd := providerCommand(
		context.Background(),
		[]string{"cmd.exe", "/c", "exit", "0"},
	)
	attr := cmd.SysProcAttr
	if attr == nil || !attr.HideWindow {
		t.Fatalf("provider subprocess is not hidden: %+v", attr)
	}
	if attr.CreationFlags&0x08000000 == 0 {
		t.Fatalf("provider subprocess is missing CREATE_NO_WINDOW: %+v", attr)
	}
}
