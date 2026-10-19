// Package youtube owns the concrete VexStream YouTube bridge process contract.
//
// It knows how to call media_bridge.py and decode provider-shaped success/failure payloads.
// It does not own Explore history, Import lifecycle, runtime installation/repair, or Library mutation.
package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Problem struct {
	SchemaVersion   string `json:"schemaVersion,omitempty"`
	Provider        string `json:"provider,omitempty"`
	Stage           string `json:"stage,omitempty"`
	Code            string `json:"code,omitempty"`
	Title           string `json:"title,omitempty"`
	Message         string `json:"message,omitempty"`
	Retryable       bool   `json:"retryable,omitempty"`
	SuggestedAction string `json:"suggestedAction,omitempty"`
	Technical       string `json:"technical,omitempty"`
	ObservedAt      string `json:"observedAt,omitempty"`
}

type Failure struct {
	Problem Problem
}

func (f *Failure) Error() string {
	if f == nil {
		return ""
	}
	return strings.TrimSpace(
		strings.Join(
			nonEmpty(f.Problem.Title, f.Problem.Message, f.Problem.SuggestedAction),
			" ",
		),
	)
}

type SearchResult struct {
	ProviderID string  `json:"id,omitempty"`
	Title      string  `json:"title,omitempty"`
	Channel    string  `json:"channel,omitempty"`
	URL        string  `json:"url,omitempty"`
	Thumbnail  string  `json:"thumbnail,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

type Section struct {
	Index    int     `json:"index"`
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist,omitempty"`
	Selected bool    `json:"selected"`
}

type AudioQuality struct {
	Codec        string  `json:"codec,omitempty"`
	Container    string  `json:"container,omitempty"`
	BitrateKbps  float64 `json:"bitrateKbps,omitempty"`
	SampleRateHz int     `json:"sampleRateHz,omitempty"`
	Channels     int     `json:"channels,omitempty"`
	Basis        string  `json:"basis,omitempty"`
}

type Inspection struct {
	ProviderID    string       `json:"id,omitempty"`
	URL           string       `json:"url,omitempty"`
	Title         string       `json:"title,omitempty"`
	Channel       string       `json:"channel,omitempty"`
	Duration      float64      `json:"duration,omitempty"`
	Thumbnail     string       `json:"thumbnail,omitempty"`
	Description   string       `json:"description,omitempty"`
	Tags          []string     `json:"tags,omitempty"`
	Categories    []string     `json:"categories,omitempty"`
	Artist        string       `json:"artist,omitempty"`
	Album         string       `json:"album,omitempty"`
	Genre         string       `json:"genre,omitempty"`
	UploadDate    string       `json:"uploadDate,omitempty"`
	Sections      []Section    `json:"sections,omitempty"`
	HasSections   bool         `json:"hasSections,omitempty"`
	SourceQuality AudioQuality `json:"sourceQuality,omitempty"`
}

type Runner func(context.Context, []string) ([]byte, error)

type Bridge struct {
	Python        string
	ResolvePython func() string
	Script        string
	Env           []string
	run           Runner
	stream        StreamRunner
}

func New(python, script string, env []string) *Bridge {
	bridge := &Bridge{
		Python: strings.TrimSpace(python),
		Script: strings.TrimSpace(script),
		Env:    append([]string(nil), env...),
	}
	bridge.run = bridge.runProcess
	bridge.stream = bridge.runStreamingProcess
	return bridge
}

func NewDynamic(
	script string,
	env []string,
	resolvePython func() string,
) *Bridge {
	bridge := New("", script, env)
	bridge.ResolvePython = resolvePython
	return bridge
}

func (b *Bridge) pythonPath() string {
	if b == nil {
		return ""
	}
	if b.ResolvePython != nil {
		if value := strings.TrimSpace(b.ResolvePython()); value != "" {
			return value
		}
	}
	return strings.TrimSpace(b.Python)
}

func (b *Bridge) Search(
	ctx context.Context,
	query string,
	limit int,
) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("YouTube search query required")
	}
	if limit <= 0 {
		limit = 8
	}

	var response SearchResponse
	if err := b.jsonCommand(
		ctx,
		"search",
		&response,
		"search",
		query,
		"--limit",
		strconv.Itoa(limit),
	); err != nil {
		return nil, err
	}
	if response.Results == nil {
		response.Results = []SearchResult{}
	}
	return response.Results, nil
}

func (b *Bridge) Inspect(
	ctx context.Context,
	url string,
) (Inspection, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return Inspection{}, errors.New("YouTube inspect URL required")
	}

	var response Inspection
	if err := b.jsonCommand(
		ctx,
		"inspect",
		&response,
		"inspect",
		url,
	); err != nil {
		return Inspection{}, err
	}
	return response, nil
}

func (b *Bridge) jsonCommand(
	ctx context.Context,
	stage string,
	target any,
	args ...string,
) error {
	python := b.pythonPath()
	if b == nil || python == "" || strings.TrimSpace(b.Script) == "" {
		return errors.New("YouTube bridge runtime is not configured")
	}
	if b.run == nil {
		return errors.New("YouTube bridge runner is unavailable")
	}

	command := append([]string{python, b.Script}, args...)
	output, err := b.run(ctx, command)
	if err != nil {
		return &Failure{
			Problem: parseProblem(output, stage),
		}
	}
	if err := json.Unmarshal(output, target); err != nil {
		return &Failure{
			Problem: parseProblem(output, stage),
		}
	}
	return nil
}

func (b *Bridge) runProcess(
	ctx context.Context,
	command []string,
) ([]byte, error) {
	if len(command) < 2 {
		return nil, errors.New("YouTube bridge command is incomplete")
	}
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.SysProcAttr = hiddenProcAttr()
	if len(b.Env) > 0 {
		cmd.Env = append([]string(nil), b.Env...)
	}
	return cmd.CombinedOutput()
}

func parseProblem(output []byte, stage string) Problem {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		var envelope struct {
			Type    string  `json:"type"`
			Problem Problem `json:"problem"`
		}
		if json.Unmarshal([]byte(lines[index]), &envelope) == nil &&
			envelope.Type == "error" {
			if envelope.Problem.Stage == "" {
				envelope.Problem.Stage = stage
			}
			if envelope.Problem.Provider == "" {
				envelope.Problem.Provider = "youtube"
			}
			return envelope.Problem
		}
	}

	return Problem{
		SchemaVersion:   "vexstream.provider-problem/v1",
		Provider:        "youtube",
		Stage:           stage,
		Code:            "YOUTUBE_PROVIDER_ERROR",
		Title:           "YouTube could not complete this request.",
		Message:         "The provider/runtime returned an unexpected error.",
		Retryable:       true,
		SuggestedAction: "Retry once. If it repeats, inspect the provider diagnostic.",
		Technical:       strings.TrimSpace(string(output)),
	}
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func (p Problem) String() string {
	return fmt.Sprintf("%s:%s", p.Provider, p.Code)
}
