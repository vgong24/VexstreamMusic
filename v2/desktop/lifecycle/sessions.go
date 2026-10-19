// Package lifecycle owns cross-process VexStream desktop identity and safe handoff mechanics.
package lifecycle

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type sessionPort struct {
	PID       int
	Port      int
	StartedAt string
}

type LiveSession struct {
	PID             int      `json:"pid"`
	Version         string   `json:"version"`
	BuildRef        string   `json:"buildRef,omitempty"`
	URL             string   `json:"url"`
	StartedAt       string   `json:"startedAt,omitempty"`
	Current         bool     `json:"current"`
	TrackCount      int      `json:"trackCount"`
	ActiveImports   int      `json:"activeImports"`
	LibraryVerified bool     `json:"libraryVerified"`
	ImportsVerified bool     `json:"importsVerified"`
	TrackRefs       []string `json:"-"`
}

type RetireCandidate struct {
	PID              int    `json:"pid"`
	Version          string `json:"version"`
	BuildRef         string `json:"buildRef,omitempty"`
	URL              string `json:"url"`
	StartedAt        string `json:"startedAt,omitempty"`
	TrackCount       int    `json:"trackCount"`
	ActiveImports    int    `json:"activeImports"`
	UniqueTrackCount int    `json:"uniqueTrackCount"`
	Eligible         bool   `json:"eligible"`
	Reason           string `json:"reason"`
}

type SessionPlan struct {
	CurrentPID int               `json:"currentPID"`
	Sessions   []LiveSession     `json:"sessions"`
	Eligible   []RetireCandidate `json:"eligible"`
	Held       []RetireCandidate `json:"held"`
	ScanError  string            `json:"scanError,omitempty"`
	Warning    string            `json:"warning"`
}

type RetireFailure struct {
	PID   int    `json:"pid"`
	Error string `json:"error"`
}

func ReviewSessions(currentPID int, currentTrackRefs []string) SessionPlan {
	ports, err := discoverCandidatePorts()
	if err != nil {
		return SessionPlan{
			CurrentPID: currentPID,
			Sessions:   []LiveSession{},
			Eligible:   []RetireCandidate{},
			Held:       []RetireCandidate{},
			ScanError:  err.Error(),
			Warning:    sessionRetirementWarning,
		}
	}

	sessions := make([]LiveSession, 0, len(ports))
	seen := map[string]bool{}
	for _, row := range ports {
		if row.PID <= 0 || row.Port <= 0 {
			continue
		}
		url := fmt.Sprintf("http://127.0.0.1:%d/", row.Port)
		if seen[url] {
			continue
		}
		seen[url] = true
		identity := Probe(url)
		if !identity.OK {
			continue
		}
		if identity.PID != 0 && identity.PID != row.PID {
			continue
		}

		refs, libraryOK := sessionTrackRefs(url)
		active, importsOK := sessionActiveImports(url)
		sessions = append(sessions, LiveSession{
			PID:             row.PID,
			Version:         identity.Version,
			BuildRef:        identity.BuildRef,
			URL:             url,
			StartedAt:       row.StartedAt,
			Current:         row.PID == currentPID,
			TrackCount:      len(refs),
			ActiveImports:   active,
			LibraryVerified: libraryOK,
			ImportsVerified: importsOK,
			TrackRefs:       refs,
		})
	}

	return ClassifySessions(currentPID, currentTrackRefs, sessions)
}

func ClassifySessions(
	currentPID int,
	currentTrackRefs []string,
	sessions []LiveSession,
) SessionPlan {
	current := map[string]bool{}
	for _, ref := range currentTrackRefs {
		ref = strings.TrimSpace(ref)
		if ref != "" {
			current[ref] = true
		}
	}

	rows := append([]LiveSession(nil), sessions...)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Current != rows[j].Current {
			return rows[i].Current
		}
		if rows[i].Version != rows[j].Version {
			return rows[i].Version > rows[j].Version
		}
		return rows[i].PID < rows[j].PID
	})

	eligible := make([]RetireCandidate, 0)
	held := make([]RetireCandidate, 0)
	for _, session := range rows {
		if session.PID == currentPID || session.Current {
			continue
		}
		candidate := RetireCandidate{
			PID:           session.PID,
			Version:       session.Version,
			BuildRef:      session.BuildRef,
			URL:           session.URL,
			StartedAt:     session.StartedAt,
			TrackCount:    session.TrackCount,
			ActiveImports: session.ActiveImports,
		}

		switch {
		case !session.ImportsVerified:
			candidate.Reason = "Could not verify whether this session still owns active imports"
		case session.ActiveImports > 0:
			candidate.Reason = fmt.Sprintf(
				"%d active import(s) are still owned by this session",
				session.ActiveImports,
			)
		case !session.LibraryVerified:
			candidate.Reason = "Could not verify this session's live Library"
		default:
			for _, ref := range session.TrackRefs {
				if ref != "" && !current[ref] {
					candidate.UniqueTrackCount++
				}
			}
			if candidate.UniqueTrackCount > 0 {
				candidate.Reason = fmt.Sprintf(
					"Holds %d track(s) not visible in the current session",
					candidate.UniqueTrackCount,
				)
			} else {
				candidate.Eligible = true
				candidate.Reason = "Idle and its visible track IDs are covered by the current session"
			}
		}

		if candidate.Eligible {
			eligible = append(eligible, candidate)
		} else {
			held = append(held, candidate)
		}
	}

	return SessionPlan{
		CurrentPID: currentPID,
		Sessions:   rows,
		Eligible:   eligible,
		Held:       held,
		Warning:    sessionRetirementWarning,
	}
}

const sessionRetirementWarning = "Retiring an older server can stop playback in that older browser tab. Sessions with active imports, unverified state, or unique live tracks are held automatically. Library folders and media files are never deleted by session retirement."

func sessionTrackRefs(baseURL string) ([]string, bool) {
	var payload struct {
		Tracks []struct {
			ID string `json:"id"`
		} `json:"tracks"`
	}
	if !getLoopbackJSON(strings.TrimRight(baseURL, "/")+"/api/library", &payload) {
		return nil, false
	}
	refs := make([]string, 0, len(payload.Tracks))
	seen := map[string]bool{}
	for _, track := range payload.Tracks {
		ref := strings.TrimSpace(track.ID)
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		refs = append(refs, ref)
	}
	return refs, true
}

func sessionActiveImports(baseURL string) (int, bool) {
	var payload map[string]json.RawMessage
	if !getLoopbackJSON(
		strings.TrimRight(baseURL, "/")+"/api/import/jobs?local=1",
		&payload,
	) {
		return 0, false
	}

	type jobs struct {
		Active []json.RawMessage `json:"active"`
	}
	if raw, ok := payload["groups"]; ok {
		var grouped jobs
		if json.Unmarshal(raw, &grouped) != nil {
			return 0, false
		}
		return len(grouped.Active), true
	}
	raw, ok := payload["active"]
	if !ok {
		return 0, false
	}
	var active []json.RawMessage
	if json.Unmarshal(raw, &active) != nil {
		return 0, false
	}
	return len(active), true
}

func getLoopbackJSON(url string, target any) bool {
	client := http.Client{Timeout: 900 * time.Millisecond}
	response, err := client.Get(url)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 8<<20))
	return decoder.Decode(target) == nil
}

func RetireEligibleSessions(
	currentPID int,
	requested []int,
	plan SessionPlan,
) ([]int, []RetireFailure) {
	allowed := map[int]RetireCandidate{}
	for _, candidate := range plan.Eligible {
		allowed[candidate.PID] = candidate
	}

	retired := make([]int, 0, len(requested))
	failed := make([]RetireFailure, 0)
	seen := map[int]bool{}
	for _, pid := range requested {
		if pid <= 0 || seen[pid] {
			continue
		}
		seen[pid] = true
		if pid == currentPID {
			failed = append(failed, RetireFailure{
				PID: pid, Error: "refusing to stop the current VexStream process",
			})
			continue
		}
		if _, ok := allowed[pid]; !ok {
			failed = append(failed, RetireFailure{
				PID: pid,
				Error: "session is no longer eligible or is held for safety",
			})
			continue
		}
		if err := stopSessionPID(pid); err != nil {
			failed = append(failed, RetireFailure{PID: pid, Error: err.Error()})
			continue
		}
		retired = append(retired, pid)
	}
	return retired, failed
}

func CurrentPID() int {
	return os.Getpid()
}
