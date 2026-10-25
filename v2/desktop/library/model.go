// Package library owns VexStream's canonical discovered-track model and its current desktop catalog projection.
// It does not own filesystem discovery, browser rendering, playback, playlists, import, or mobile projection.
package library

type Track struct {
	ID                string   `json:"id"`
	Path              string   `json:"path"`
	Filename          string   `json:"filename"`
	Title             string   `json:"title"`
	Artist            string   `json:"artist"`
	Album             string   `json:"album"`
	Genre             string   `json:"genre"`
	Year              string   `json:"year"`
	Duration          float64  `json:"duration"`
	Source            string   `json:"source"`
	ProviderID        string   `json:"providerId,omitempty"`
	SourceURL         string   `json:"sourceUrl,omitempty"`
	Channel           string   `json:"channel,omitempty"`
	ArtistBasis       string   `json:"artistBasis,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Categories        []string `json:"categories,omitempty"`
	Description       string   `json:"description,omitempty"`
	ThumbnailURL      string   `json:"thumbnailUrl,omitempty"`
EmbeddedArtwork  bool     `json:"embeddedArtwork,omitempty"`
	IndexedAt         string   `json:"indexedAt,omitempty"`
	AddedAt           string   `json:"addedAt"`
	SourceTitle       string   `json:"sourceTitle,omitempty"`
	SourceArtist      string   `json:"sourceArtist,omitempty"`
	SourceAlbum       string   `json:"sourceAlbum,omitempty"`
	SourceGenre       string   `json:"sourceGenre,omitempty"`
	OverlayUpdatedAt  string   `json:"overlayUpdatedAt,omitempty"`
	PlaybackStart     float64  `json:"playbackStart,omitempty"`
	PlaybackEnd       *float64 `json:"playbackEnd,omitempty"`
	PlaybackUpdatedAt string   `json:"playbackUpdatedAt,omitempty"`
}

type Overlay struct {
	Title     *string `json:"title,omitempty"`
	Artist    *string `json:"artist,omitempty"`
	Album     *string `json:"album,omitempty"`
	Genre     *string `json:"genre,omitempty"`
	UpdatedAt string  `json:"updatedAt"`
}

type PlaybackWindow struct {
	Start     float64  `json:"start,omitempty"`
	End       *float64 `json:"end,omitempty"`
	UpdatedAt string   `json:"updatedAt"`
}

type CacheEntry struct {
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	ModTime      int64  `json:"modTime"`
	CatalogStamp int64  `json:"catalogStamp"`
	Track        Track  `json:"track"`
}

const CacheSchemaV3 = "vexstream.library-index/v3"
