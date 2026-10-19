package v4app

import (
	"strings"
	"testing"
)

func TestEmbeddedV4AssetsCarryBrowserAndProviderBridge(t *testing.T) {
	index, err := Assets.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "VexStream Music") {
		t.Fatal("embedded V4 browser entry missing")
	}
	bridge, err := MediaBridgeBytes()
	if err != nil {
		t.Fatal(err)
	}
	text := string(bridge)
	if !strings.Contains(text, `VERSION = "4.0.0"`) ||
		!strings.Contains(text, "vexstream.provider-problem/v1") ||
		!strings.Contains(text, "progress=10+int(ratio*30)") ||
		!strings.Contains(text, "Download complete — preparing audio") {
		t.Fatal("embedded V4 provider bridge identity/progress drift")
	}
}
