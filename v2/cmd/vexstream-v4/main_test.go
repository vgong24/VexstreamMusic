package main

import (
	"strings"
	"testing"
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
