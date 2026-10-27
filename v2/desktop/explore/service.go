package explore

import (
	"context"
	"errors"
	"strings"
	"time"
)

const SchemaVersion = "vexstream.discovery-candidates/v2"

type Provider interface {
	Search(context.Context, string, int) ([]Candidate, error)
}

type Matcher interface {
	Match(Candidate) LibraryMatch
}

type Frame struct {
	SchemaVersion string      `json:"schemaVersion"`
	Seed          Seed        `json:"seed"`
	Basis         Basis       `json:"basis"`
	Candidates    []Candidate `json:"candidates"`
	GeneratedAt   string      `json:"generatedAt"`
}

type Basis struct {
	Provider   string  `json:"provider"`
	Method     string  `json:"method"`
	Queries    []Query `json:"queries"`
	CreatorCap int     `json:"creatorCap"`
	Selection  string  `json:"selection"`
}

type Service struct {
	Provider    Provider
	Matcher     Matcher
	ProviderRef string
	Limit       int
	CreatorCap  int
	PerQuery    int
	Now         func() time.Time
}

func (s Service) Explore(ctx context.Context, seed Seed) (Frame, error) {
	seed.Title = strings.TrimSpace(seed.Title)
	if seed.Title == "" {
		return Frame{}, errors.New("explore seed title required")
	}
	if s.Provider == nil {
		return Frame{}, errors.New("explore provider required")
	}
	queries := Queries(seed)
	if len(queries) == 0 {
		return Frame{}, errors.New("explore query formation failed")
	}

	perQuery := s.PerQuery
	if perQuery <= 0 {
		perQuery = 6
	}
	limit := s.Limit
	if limit <= 0 {
		limit = 16
	}
	creatorCap := s.CreatorCap
	if creatorCap <= 0 {
		creatorCap = 2
	}
	providerRef := strings.TrimSpace(s.ProviderRef)
	if providerRef == "" {
		providerRef = "provider"
	}
	now := s.Now
	if now == nil {
		now = time.Now
	}

	pools := make([][]Candidate, 0, len(queries))
	for _, query := range queries {
		rows, err := s.Provider.Search(ctx, query.Query, perQuery)
		if err != nil {
			return Frame{}, err
		}
		pool := make([]Candidate, 0, len(rows))
		seen := map[string]bool{}
		for _, row := range rows {
			row.ProviderID = strings.TrimSpace(row.ProviderID)
			row.Title = strings.TrimSpace(row.Title)
			row.Channel = strings.TrimSpace(row.Channel)
			row.URL = strings.TrimSpace(row.URL)
			row.Thumbnail = strings.TrimSpace(row.Thumbnail)
			if row.Title == "" || SameSeed(seed, row) {
				continue
			}
			key := candidateKey(row)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			row.Axis = query.Axis
			row.AxisLabel = query.Label
			row.Query = query.Query
			row.LibraryMatch = LibraryMatch{State: "NOT_IN_LIBRARY"}
			if s.Matcher != nil {
				row.LibraryMatch = s.Matcher.Match(row)
				if strings.TrimSpace(row.LibraryMatch.State) == "" {
					row.LibraryMatch.State = "NOT_IN_LIBRARY"
				}
			}
			pool = append(pool, row)
		}
		pools = append(pools, pool)
	}

	return Frame{
		SchemaVersion: SchemaVersion,
		Seed:          seed,
		Basis: Basis{
			Provider:   providerRef,
			Method:     "SEARCH_DERIVED_MULTI_AXIS",
			Queries:    append([]Query(nil), queries...),
			CreatorCap: creatorCap,
			Selection:  "ROUND_ROBIN",
		},
		Candidates:  SelectCandidates(pools, limit, creatorCap),
		GeneratedAt: now().UTC().Format(time.RFC3339Nano),
	}, nil
}
