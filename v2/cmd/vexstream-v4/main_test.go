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
			OK:       true,
			Version:  "3.0.0",
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


func TestDifferentV4BuildConflictExplainsExactSingleWriterRequirement(t *testing.T) {
	err := differentV4BuildConflict(
		lifecycle.LiveClaim{
			URL: "http://127.0.0.1:4321/",
			Identity: lifecycle.Identity{
				OK:       true,
				Version:  "4.0.0",
				BuildRef: "v4-old.bin-1111111111111111",
			},
		},
		"v4-new.bin-2222222222222222",
	)
	if err == nil ||
		!strings.Contains(err.Error(), "v4-old.bin-1111111111111111") ||
		!strings.Contains(err.Error(), "v4-new.bin-2222222222222222") ||
		!strings.Contains(err.Error(), "Exit the running V4") ||
		!strings.Contains(err.Error(), "shared Library state") {
		t.Fatalf("different V4 build conflict message drift: %v", err)
	}
}
