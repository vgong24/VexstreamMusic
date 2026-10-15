package runtimeenv

import (
	"testing"
)

func TestRuntimeServiceCachesOneCurrentProjection(t *testing.T) {
	service := NewService(t.TempDir())
	first := service.Status()
	second := service.Status()
	if first.SearchReady != second.SearchReady ||
		first.ImportReady != second.ImportReady ||
		first.Python != second.Python {
		t.Fatalf("runtime cached projection drift: first=%+v second=%+v", first, second)
	}
	if service.Python() != first.Python {
		t.Fatalf("runtime Python projection drift: %q != %q", service.Python(), first.Python)
	}
}
