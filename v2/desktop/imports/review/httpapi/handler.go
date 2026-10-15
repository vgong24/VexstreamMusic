// Package httpapi exposes the provider-inspection + advisory duplicate-review boundary.
//
// This endpoint does not download, stage, commit, mutate Library, or decide that an advisory match
// blocks an import.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/duplicates"
	importprovider "github.com/vgong24/VexstreamMusic/v2/desktop/imports/provider"
	"github.com/vgong24/VexstreamMusic/v2/desktop/imports/review"
	youtubeprovider "github.com/vgong24/VexstreamMusic/v2/desktop/provider/youtube"
)

type Reviewer interface {
	InspectAndReview(context.Context, string) (review.Result, error)
}

type Handler struct {
	reviewer Reviewer
}

func New(reviewer Reviewer) *Handler {
	if reviewer == nil {
		panic("Import review HTTP adapter requires reviewer")
	}
	return &Handler{reviewer: reviewer}
}

type request struct {
	URL string `json:"url"`
}

type response struct {
	Inspection inspectionResponse `json:"inspection"`
	Items      []duplicates.Item  `json:"items"`
	MatchCount int                `json:"matchCount"`
	Advisory   bool               `json:"advisory"`
}

type inspectionResponse struct {
	ProviderID    string                   `json:"id,omitempty"`
	URL           string                   `json:"url,omitempty"`
	Title         string                   `json:"title,omitempty"`
	Channel       string                   `json:"channel,omitempty"`
	Duration      float64                  `json:"duration,omitempty"`
	Thumbnail     string                   `json:"thumbnail,omitempty"`
	Description   string                   `json:"description,omitempty"`
	Tags          []string                 `json:"tags,omitempty"`
	Categories    []string                 `json:"categories,omitempty"`
	Artist        string                   `json:"artist,omitempty"`
	Album         string                   `json:"album,omitempty"`
	Genre         string                   `json:"genre,omitempty"`
	UploadDate    string                   `json:"uploadDate,omitempty"`
	Sections      []sectionResponse        `json:"sections,omitempty"`
	HasSections   bool                     `json:"hasSections,omitempty"`
	SourceQuality duplicates.AudioQuality  `json:"sourceQuality,omitempty"`
}

type sectionResponse struct {
	Index    int     `json:"index"`
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist,omitempty"`
	Selected bool    `json:"selected"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/import/inspect" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var input request
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid import inspection request", http.StatusBadRequest)
		return
	}
	input.URL = strings.TrimSpace(input.URL)
	if input.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	result, err := h.reviewer.InspectAndReview(r.Context(), input.URL)
	if err != nil {
		var providerFailure *youtubeprovider.Failure
		if errors.As(err, &providerFailure) {
			jsonOut(w, http.StatusBadGateway, map[string]any{
				"error":   providerFailure.Error(),
				"problem": providerFailure.Problem,
			})
			return
		}
		jsonOut(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	jsonOut(w, http.StatusOK, response{
		Inspection: projectInspection(result.Inspection),
		Items:      append([]duplicates.Item(nil), result.Items...),
		MatchCount: result.MatchCount,
		Advisory:   result.Advisory,
	})
}

func projectInspection(value importprovider.Inspection) inspectionResponse {
	sections := make([]sectionResponse, 0, len(value.Sections))
	for _, section := range value.Sections {
		sections = append(sections, sectionResponse{
			Index: section.Index,
			Start: section.Start,
			End: section.End,
			Title: section.Title,
			Artist: section.Artist,
			Selected: section.Selected,
		})
	}
	return inspectionResponse{
		ProviderID: value.ProviderID,
		URL: value.URL,
		Title: value.Title,
		Channel: value.Channel,
		Duration: value.Duration,
		Thumbnail: value.Thumbnail,
		Description: value.Description,
		Tags: append([]string(nil), value.Tags...),
		Categories: append([]string(nil), value.Categories...),
		Artist: value.Artist,
		Album: value.Album,
		Genre: value.Genre,
		UploadDate: value.UploadDate,
		Sections: sections,
		HasSections: value.HasSections,
		SourceQuality: value.SourceQuality,
	}
}

func jsonOut(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
