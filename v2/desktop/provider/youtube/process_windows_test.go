//go:build windows

package youtube

import "testing"

func TestProviderSubprocessStaysHiddenOnWindows(t *testing.T) {
	attr := hiddenProcAttr()
	if attr == nil || !attr.HideWindow {
		t.Fatalf("provider subprocess is not hidden: %+v", attr)
	}
	if attr.CreationFlags&0x08000000 == 0 {
		t.Fatalf("provider subprocess is missing CREATE_NO_WINDOW: %+v", attr)
	}
}
