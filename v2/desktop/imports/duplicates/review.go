// Package duplicates owns advisory Import duplicate-review scoring.
//
// DUPLICATE_ADVISORY != IMPORT_BLOCK.
// This package can compare prepared provider metadata with current Library tracks, but it never
// mutates Library, rejects an import, or claims two recordings are definitively identical.
package duplicates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
	"github.com/vgong24/VexstreamMusic/v2/desktop/musictext"
)

type AudioQuality struct {
	Codec        string  `json:"codec,omitempty"`
	Container    string  `json:"container,omitempty"`
	BitrateKbps  float64 `json:"bitrateKbps,omitempty"`
	SampleRateHz int     `json:"sampleRateHz,omitempty"`
	Channels     int     `json:"channels,omitempty"`
	Basis        string  `json:"basis,omitempty"`
}

type QualityComparison struct {
	State   string `json:"state"`
	Summary string `json:"summary"`
}

type Match struct {
	TrackRef          string            `json:"trackId"`
	Title             string            `json:"title"`
	Artist            string            `json:"artist"`
	Album             string            `json:"album"`
	Duration          float64           `json:"duration"`
	ThumbnailURL      string            `json:"thumbnailUrl,omitempty"`
	ProviderID        string            `json:"providerId,omitempty"`
	SourceURL         string            `json:"sourceUrl,omitempty"`
	Score             int               `json:"score"`
	Confidence        string            `json:"confidence"`
	Relation          string            `json:"relation"`
	Reasons           []string          `json:"reasons"`
	LocalQuality      AudioQuality      `json:"localQuality"`
	QualityComparison QualityComparison `json:"qualityComparison"`
}

type Incoming struct {
	ProviderID    string       `json:"id,omitempty"`
	URL           string       `json:"url,omitempty"`
	Title         string       `json:"title,omitempty"`
	Artist        string       `json:"artist,omitempty"`
	SourceCreator string       `json:"channel,omitempty"`
	Duration      float64      `json:"duration,omitempty"`
	SourceQuality AudioQuality `json:"sourceQuality,omitempty"`
}

type Section struct {
	Index int     `json:"index"`
	Title string  `json:"title"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type ReviewRequest struct {
	Incoming
	Sections []Section `json:"sections,omitempty"`
}

type Item struct {
	Kind          string       `json:"kind"`
	Index         int          `json:"index,omitempty"`
	Title         string       `json:"title"`
	Duration      float64      `json:"duration"`
	SourceQuality AudioQuality `json:"sourceQuality"`
	Matches       []Match      `json:"matches"`
}

func Review(request ReviewRequest, tracks []library.Track) []Item {
	items := []Item{}
	if matches := Find(request.Incoming, tracks); len(matches) > 0 {
		items = append(items, Item{
			Kind: "whole", Title: request.Title, Duration: request.Duration,
			SourceQuality: request.SourceQuality, Matches: matches,
		})
	}
	for _, section := range request.Sections {
		duration := section.End - section.Start
		if duration < 0 {
			duration = 0
		}
		incoming := request.Incoming
		incoming.Title = section.Title
		incoming.ProviderID = ""
		incoming.URL = ""
		incoming.Duration = duration
		if matches := Find(incoming, tracks); len(matches) > 0 {
			items = append(items, Item{
				Kind: "section", Index: section.Index, Title: section.Title, Duration: duration,
				SourceQuality: request.SourceQuality, Matches: matches,
			})
		}
	}
	return items
}

func Find(incoming Incoming, tracks []library.Track) []Match {
	matches := make([]Match, 0)
	for _, track := range tracks {
		score, reasons, relation := Score(incoming, track)
		if score < 55 {
			continue
		}
		artist := track.Artist
		if artist == "" {
			artist = track.Channel
		}
		localQuality := EstimateLocalQuality(track)
		matches = append(matches, Match{
			TrackRef: track.ID, Title: track.Title, Artist: artist, Album: track.Album,
			Duration: track.Duration, ThumbnailURL: track.ThumbnailURL,
			ProviderID: track.ProviderID, SourceURL: track.SourceURL,
			Score: score, Confidence: Confidence(score), Relation: relation,
			Reasons: append([]string(nil), reasons...), LocalQuality: localQuality,
			QualityComparison: CompareQuality(incoming.SourceQuality, localQuality),
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return strings.ToLower(matches[i].Title) < strings.ToLower(matches[j].Title)
		}
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > 4 {
		matches = matches[:4]
	}
	return matches
}

func Score(incoming Incoming, track library.Track) (int, []string, string) {
	score := 0
	reasons := []string{}
	relation := "POSSIBLE_MATCH"

	if incoming.ProviderID != "" && track.ProviderID != "" && incoming.ProviderID == track.ProviderID {
		score += 100
		reasons = append(reasons, "same provider item")
		relation = "SAME_SOURCE"
	}
	if incoming.URL != "" && track.SourceURL != "" && strings.EqualFold(incoming.URL, track.SourceURL) {
		score += 100
		reasons = append(reasons, "same source URL")
		relation = "SAME_SOURCE"
	}

	similarity, exactTitle := maxTitleSimilarity(incoming, track)
	if exactTitle {
		score += 55
		reasons = append(reasons, "same normalized song title")
	} else if similarity >= 0.82 {
		score += 42
		reasons = append(reasons, "very similar title")
	} else if similarity >= 0.62 {
		score += 27
		reasons = append(reasons, "similar title")
	}

	existingArtist := track.Artist
	if existingArtist == "" {
		existingArtist = track.Channel
	}
	if incoming.Artist != "" && existingArtist != "" {
		if musictext.Normalize(incoming.Artist) == musictext.Normalize(existingArtist) {
			score += 20
			reasons = append(reasons, "same artist/creator")
		} else if musictext.Similarity(incoming.Artist, existingArtist) >= 0.7 {
			score += 10
			reasons = append(reasons, "similar artist/creator")
		}
	}
	if incoming.SourceCreator != "" &&
		existingArtist != "" &&
		musictext.Normalize(incoming.SourceCreator) == musictext.Normalize(existingArtist) {
		score += 18
		reasons = append(reasons, "source creator matches existing artist")
	}

	if incoming.Duration > 0 && track.Duration > 0 {
		diff := incoming.Duration - track.Duration
		if diff < 0 {
			diff = -diff
		}
		switch {
		case diff <= 3:
			score += 22
			reasons = append(reasons, "duration within 3s")
		case diff <= 10:
			score += 13
			reasons = append(reasons, "duration within 10s")
		case diff <= 30:
			score += 5
			reasons = append(reasons, "similar duration")
		case exactTitle:
			score -= 6
			reasons = append(reasons, fmt.Sprintf("duration differs by %ds; may be a different edit", int(diff+0.5)))
		}
		if relation != "SAME_SOURCE" && exactTitle {
			if diff <= 10 {
				relation = "LIKELY_SAME_RECORDING"
			} else {
				relation = "SAME_SONG_DIFFERENT_EDIT_POSSIBLE"
			}
		}
	}
	if relation == "POSSIBLE_MATCH" && exactTitle {
		relation = "SAME_SONG_POSSIBLE"
	}
	return score, reasons, relation
}

func Confidence(score int) string {
	switch {
	case score >= 95:
		return "very likely"
	case score >= 75:
		return "likely"
	default:
		return "possible"
	}
}

func EstimateLocalQuality(track library.Track) AudioQuality {
	codec := strings.TrimPrefix(strings.ToLower(filepath.Ext(track.Path)), ".")
	if codec == "" {
		codec = "unknown"
	}
	quality := AudioQuality{
		Codec: codec, Container: codec, Basis: "local.file-size-duration-estimate",
	}
	if track.Duration > 0 {
		if stat, err := os.Stat(track.Path); err == nil && stat.Size() > 0 {
			quality.BitrateKbps = float64(stat.Size()) * 8 / 1000 / track.Duration
		}
	}
	return quality
}

func CompareQuality(source, local AudioQuality) QualityComparison {
	if source.BitrateKbps <= 0 || local.BitrateKbps <= 0 {
		return QualityComparison{
			State:   "INSUFFICIENT_DATA",
			Summary: "Not enough format data to compare audio quality before download.",
		}
	}
	sourceFamily, localFamily := codecFamily(source.Codec), codecFamily(local.Codec)
	if sourceFamily == "" || localFamily == "" ||
		sourceFamily == "unknown" || localFamily == "unknown" ||
		sourceFamily != localFamily {
		return QualityComparison{
			State:   "NOT_DIRECTLY_COMPARABLE",
			Summary: "Different codecs or containers; bitrate alone cannot prove which copy sounds better.",
		}
	}
	ratio := source.BitrateKbps / local.BitrateKbps
	switch {
	case ratio >= 0.88 && ratio <= 1.12:
		return QualityComparison{
			State:   "SIMILAR_INDICATORS",
			Summary: "The available codec/bitrate indicators are in a similar range.",
		}
	case ratio > 1.12:
		return QualityComparison{
			State:   "SOURCE_HIGHER_BITRATE_SAME_CODEC",
			Summary: "The source reports a higher bitrate in the same codec family; listening is still the final quality check.",
		}
	default:
		return QualityComparison{
			State:   "LOCAL_HIGHER_BITRATE_SAME_CODEC",
			Summary: "Your existing local copy has the higher bitrate in the same codec family; listening is still the final quality check.",
		}
	}
}

func maxTitleSimilarity(incoming Incoming, track library.Track) (float64, bool) {
	left := musictext.TitleVariants(incoming.Title, incoming.Artist, incoming.SourceCreator)
	right := musictext.TitleVariants(track.Title, track.Artist, track.Channel)
	best := 0.0
	exact := false
	for _, a := range left {
		for _, b := range right {
			if a == b || (musictext.Compact(a) != "" && musictext.Compact(a) == musictext.Compact(b)) {
				exact = true
				best = 1
				continue
			}
			if similarity := musictext.Similarity(a, b); similarity > best {
				best = similarity
			}
		}
	}
	return best, exact
}

func codecFamily(codec string) string {
	codec = strings.ToLower(strings.TrimSpace(codec))
	switch {
	case strings.Contains(codec, "mp3") || strings.Contains(codec, "mp3lame"):
		return "mp3"
	case strings.Contains(codec, "opus"):
		return "opus"
	case strings.Contains(codec, "aac") || strings.Contains(codec, "mp4a"):
		return "aac"
	case strings.Contains(codec, "vorbis"):
		return "vorbis"
	default:
		return codec
	}
}
