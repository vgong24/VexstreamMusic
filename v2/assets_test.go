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
	app, err := Assets.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := Assets.ReadFile("web/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	indexText := string(index)
	appText := string(app)
	styleText := string(styles)
	if !strings.Contains(indexText, "VexStream Music") ||
		!strings.Contains(indexText, `id="importReviewLoading"`) ||
		!strings.Contains(indexText, `id="importDuplicateDetails"`) ||
		!strings.Contains(appText, "async function loadImportReview") ||
		!strings.Contains(appText, "function importDuplicateReviewHTML") ||
		!strings.Contains(styleText, ".import-review-spinner") ||
		!strings.Contains(styleText, ".import-duplicate-review") {
		t.Fatal("embedded V4 browser/import product contract drift")
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
