package librarymatch

import (
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/explore"
	"github.com/vgong24/VexstreamMusic/v2/desktop/library"
)

type ProjectionFunc func() library.Projection

type SeedResolver struct {
	projection ProjectionFunc
}

func NewSeedResolver(projection ProjectionFunc) *SeedResolver {
	if projection == nil {
		panic("Explore seed resolver requires Library projection")
	}
	return &SeedResolver{projection: projection}
}

func (r *SeedResolver) ResolveTrackSeed(
	trackRef string,
) (explore.Seed, bool) {
	trackRef = strings.TrimSpace(trackRef)
	if trackRef == "" {
		return explore.Seed{}, false
	}
	for _, track := range r.projection().Tracks {
		if track.ID != trackRef {
			continue
		}
		artist := strings.TrimSpace(track.Artist)
		if artist == "" {
			artist = strings.TrimSpace(track.SourceArtist)
		}
		return explore.Seed{
			Kind:       "library",
			TrackRef:   track.ID,
			ProviderID: track.ProviderID,
			URL:        track.SourceURL,
			Title:      track.Title,
			Artist:     artist,
			Album:      track.Album,
			Genre:      track.Genre,
			Year:       track.Year,
			Channel:    track.Channel,
			Duration:   track.Duration,
		}, true
	}
	return explore.Seed{}, false
}
