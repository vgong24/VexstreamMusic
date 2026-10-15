// Package musictext owns VexStream's provider-independent normalization used when comparing
// human/provider music labels. It is intentionally not metadata authority or fuzzy-match policy.
package musictext

import (
	"strings"
	"unicode"
)

var noisePhrases = []string{
	"official video",
	"official audio",
	"lyrics",
	"lyric video",
	"music video",
	"hd",
	"4k",
	"remastered",
	"visualizer",
	"full album",
	"playlist",
}

func Normalize(value string) string {
	value = strings.ToLower(value)
	for _, noise := range noisePhrases {
		value = strings.ReplaceAll(value, noise, " ")
	}
	var out strings.Builder
	space := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			space = false
		} else if !space {
			out.WriteByte(' ')
			space = true
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

func Compact(value string) string {
	return strings.ReplaceAll(Normalize(value), " ", "")
}

func Tokens(value string) map[string]bool {
	out := map[string]bool{}
	for _, token := range strings.Fields(Normalize(value)) {
		if len([]rune(token)) >= 2 {
			out[token] = true
		}
	}
	return out
}

func TitleVariants(title string, prefixes ...string) []string {
	base := Normalize(title)
	if base == "" {
		return nil
	}
	out := []string{base}
	seen := map[string]bool{base: true}
	for _, prefix := range prefixes {
		prefix = Normalize(prefix)
		if prefix == "" || !strings.HasPrefix(base, prefix+" ") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(base, prefix))
		if len(strings.Fields(rest)) < 2 || seen[rest] {
			continue
		}
		seen[rest] = true
		out = append(out, rest)
	}
	return out
}

func Similarity(left, right string) float64 {
	a, b := Tokens(left), Tokens(right)
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection, union := 0, len(a)
	for token := range b {
		if a[token] {
			intersection++
		} else {
			union++
		}
	}
	return float64(intersection) / float64(union)
}
