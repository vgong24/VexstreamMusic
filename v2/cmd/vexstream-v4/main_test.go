package main

import (
	"strings"
	"testing"

	"github.com/vgong24/VexstreamMusic/v2/desktop/lifecycle"
)

func TestEffectiveBuildRefIncludesExactBinaryIdentity(t *testing.T) {
	original := buildRef
	defer func() { buildRef = original }()
	buildRef = "source-test"

	first, err := effectiveBuildRef()
	if err != nil {
		t.Fatal(err)
	}
	second, err := effectiveBuildRef()
	if err != nil {
		t.Fatal(err)
	}
	if first != second ||
		!strings.HasPrefix(first, "source-test.bin-") ||
		len(strings.TrimPrefix(first, "source-test.bin-")) != 16 {
		t.Fatalf("binary build identity drift: first=%q second=%q", first, second)
	}
}


func TestProtectedDesktopConflictExplainsSingleWriterRequirement(t *testing.T) {
	err := protectedDesktopConflict(lifecycle.LiveClaim{
		URL: "http://127.0.0.1:1234/",
		Identity: lifecycle.Identity{
			OK: true,
			Version: "3.0.0",
			BuildRef: "v3-build",
		},
	})
	if err == nil ||
		!strings.Contains(err.Error(), "3.0.0") ||
		!strings.Contains(err.Error(), "shared Library state") ||
		!strings.Contains(err.Error(), "Exit the protected desktop") {
		t.Fatalf("protected desktop conflict message drift: %v", err)
	}
}
