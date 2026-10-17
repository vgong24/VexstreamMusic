package musictext

import (
	"reflect"
	"testing"
)

func TestNormalizePreservesV3MusicNoiseLaw(t *testing.T) {
	got := Normalize("Artist — Song (Official Video) [4K] Lyrics")
	if got != "artist song" {
		t.Fatalf("normalize drift: %q", got)
	}
}

func TestTitleVariantsCanRemoveKnownCreatorPrefix(t *testing.T) {
	got := TitleVariants("Artist Song Name Live", "Artist")
	want := []string{"artist song name live", "song name live"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("variants drift: got=%v want=%v", got, want)
	}
}

func TestTokensIgnoreOneRuneFragmentsAndSimilarityUsesJaccard(t *testing.T) {
	if got := Tokens("a big x song"); !reflect.DeepEqual(got, map[string]bool{"big": true, "song": true}) {
		t.Fatalf("tokens drift: %+v", got)
	}
	if got := Similarity("blue dream song", "blue song remix"); got != 0.5 {
		t.Fatalf("similarity drift: %v", got)
	}
}

func TestCompactIgnoresFormattingNoise(t *testing.T) {
	if Compact("Song Name") != Compact("song-name Official Audio") {
		t.Fatal("compact normalized identity drift")
	}
}
