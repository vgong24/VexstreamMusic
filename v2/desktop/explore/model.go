// Package explore owns provider-independent VexStream Explore query formation and candidate selection.
//
// It does not own the temporary browser session, provider process/runtime, Library mutation,
// preview playback, import/acquisition, or AI recommendation semantics.
package explore

import "strings"

type Seed struct {
	Kind       string  `json:"kind,omitempty"`
	TrackRef   string  `json:"trackId,omitempty"`
	ProviderID string  `json:"providerId,omitempty"`
	URL        string  `json:"url,omitempty"`
	Title      string  `json:"title,omitempty"`
	Artist     string  `json:"artist,omitempty"`
	Album      string  `json:"album,omitempty"`
	Genre      string  `json:"genre,omitempty"`
	Year       string  `json:"year,omitempty"`
	Channel    string  `json:"channel,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
}

type Query struct {
	Axis  string `json:"axis"`
	Label string `json:"label"`
	Query string `json:"query"`
}

type LibraryMatch struct {
	State      string `json:"state"`
	TrackRef   string `json:"trackId,omitempty"`
	Title      string `json:"title,omitempty"`
	Artist     string `json:"artist,omitempty"`
	Confidence string `json:"confidence,omitempty"`
	Relation   string `json:"relation,omitempty"`
	Score      int    `json:"score,omitempty"`
}

type Candidate struct {
	ProviderID   string       `json:"providerId,omitempty"`
	Title        string       `json:"title"`
	Channel      string       `json:"channel,omitempty"`
	URL          string       `json:"url,omitempty"`
	Thumbnail    string       `json:"thumbnail,omitempty"`
	Duration     float64      `json:"duration,omitempty"`
	Axis         string       `json:"axis,omitempty"`
	AxisLabel    string       `json:"axisLabel,omitempty"`
	Query        string       `json:"query,omitempty"`
	LibraryMatch LibraryMatch `json:"libraryMatch"`
}

func Queries(seed Seed) []Query {
	artist := firstNonEmpty(seed.Artist, seed.Channel)
	title := strings.TrimSpace(seed.Title)
	album := strings.TrimSpace(seed.Album)
	genre := strings.TrimSpace(seed.Genre)
	year := strings.TrimSpace(seed.Year)

	rows := []Query{}
	switch {
	case artist != "" && album != "":
		rows = appendQuery(rows, "CLOSER", "Closer", artist+" "+album)
	case artist != "" && genre != "":
		rows = appendQuery(rows, "CLOSER", "Closer", artist+" "+genre)
	case artist != "":
		rows = appendQuery(rows, "CLOSER", "Closer", artist+" music")
	case title != "":
		rows = appendQuery(rows, "CLOSER", "Closer", title)
	}

	if genre != "" {
		rows = appendQuery(rows, "NEIGHBORHOOD", "Neighborhood", genre+" music")
	} else if year != "" {
		rows = appendQuery(rows, "NEIGHBORHOOD", "Neighborhood", year+" music")
	}

	if year != "" && genre != "" {
		rows = appendQuery(rows, "ERA", "Same era", year+" "+genre+" music")
	} else if year != "" {
		rows = appendQuery(rows, "ERA", "Same era", year+" music")
	}

	if title != "" {
		rows = appendQuery(rows, "VERSIONS", "Versions", title+" cover remix live")
	}

	return rows
}

func SelectCandidates(pools [][]Candidate, limit, creatorCap int) []Candidate {
	if limit <= 0 {
		return nil
	}
	if creatorCap <= 0 {
		creatorCap = limit
	}

	positions := make([]int, len(pools))
	seen := map[string]bool{}
	creatorCounts := map[string]int{}
	selected := []Candidate{}

	for len(selected) < limit {
		addedRound := false
		for poolIndex, pool := range pools {
			for positions[poolIndex] < len(pool) {
				candidate := pool[positions[poolIndex]]
				positions[poolIndex]++

				key := candidateKey(candidate)
				if key == "" || seen[key] {
					continue
				}
				creator := normalizeMusicText(candidate.Channel)
				if creator != "" && creatorCounts[creator] >= creatorCap {
					continue
				}

				seen[key] = true
				if creator != "" {
					creatorCounts[creator]++
				}
				selected = append(selected, candidate)
				addedRound = true
				break
			}
			if len(selected) >= limit {
				break
			}
		}
		if !addedRound {
			break
		}
	}

	return selected
}

func SameSeed(seed Seed, candidate Candidate) bool {
	if seed.ProviderID != "" &&
		candidate.ProviderID != "" &&
		strings.EqualFold(seed.ProviderID, candidate.ProviderID) {
		return true
	}
	if seed.URL != "" &&
		candidate.URL != "" &&
		strings.EqualFold(seed.URL, candidate.URL) {
		return true
	}
	return normalizeMusicText(seed.Title) != "" &&
		normalizeMusicText(seed.Title) == normalizeMusicText(candidate.Title) &&
		normalizeMusicText(firstNonEmpty(seed.Artist, seed.Channel)) ==
			normalizeMusicText(candidate.Channel)
}

func appendQuery(rows []Query, axis, label, query string) []Query {
	query = strings.TrimSpace(query)
	if query == "" {
		return rows
	}
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Query), query) {
			return rows
		}
	}
	return append(rows, Query{Axis: axis, Label: label, Query: query})
}

func candidateKey(candidate Candidate) string {
	if id := strings.ToLower(strings.TrimSpace(candidate.ProviderID)); id != "" {
		return "provider:" + id
	}
	if value := strings.ToLower(strings.TrimSpace(candidate.URL)); value != "" {
		return "url:" + value
	}
	title := normalizeMusicText(candidate.Title)
	channel := normalizeMusicText(candidate.Channel)
	if title == "" && channel == "" {
		return ""
	}
	return "text:" + title + "|" + channel
}

func normalizeMusicText(value string) string {
	value = strings.ToLower(value)
	for _, noise := range []string{
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
	} {
		value = strings.ReplaceAll(value, noise, " ")
	}
	var out strings.Builder
	space := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			out.WriteRune(r)
			space = false
		} else if !space {
			out.WriteByte(' ')
			space = true
		}
	}
	return strings.Join(strings.Fields(out.String()), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
