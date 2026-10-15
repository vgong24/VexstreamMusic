package youtube

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var ErrPrepareCancelled = errors.New("YouTube import preparation cancelled")

type PrepareSection struct {
	Index       int     `json:"index"`
	SourceTitle string  `json:"sourceTitle,omitempty"`
	Title       string  `json:"title,omitempty"`
	Artist      string  `json:"artist,omitempty"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Selected    bool    `json:"selected"`
}

type PreparePlan struct {
	URL              string           `json:"url"`
	Destination      string           `json:"destination"`
	StagingDirectory string           `json:"stagingDirectory"`
	CancelPath       string           `json:"cancelPath,omitempty"`
	Mode             string           `json:"mode,omitempty"`
	WholeTitle       string           `json:"wholeTitle,omitempty"`
	ArtistOverride   string           `json:"artistOverride,omitempty"`
	Collection       string           `json:"collection,omitempty"`
	Genre            string           `json:"genre,omitempty"`
	Sections         []PrepareSection `json:"sections,omitempty"`
}

type PreparedOutput struct {
	RelativePath string         `json:"relativePath"`
	Filename     string         `json:"filename"`
	CatalogEntry map[string]any `json:"catalogEntry"`
}

type PrepareResult struct {
	Outputs []PreparedOutput `json:"outputs"`
	Staged  bool             `json:"staged"`
}

type PrepareEvent struct {
	Type     string
	Stage    string
	Progress int
	Message  string
	Problem  *Problem
}

type StreamRunner func(
	context.Context,
	[]string,
	func([]byte),
	func([]byte),
) error

func (b *Bridge) Prepare(
	ctx context.Context,
	plan PreparePlan,
	onEvent func(PrepareEvent),
) (PrepareResult, error) {
	if b == nil ||
		strings.TrimSpace(b.Python) == "" ||
		strings.TrimSpace(b.Script) == "" {
		return PrepareResult{}, errors.New("YouTube bridge runtime is not configured")
	}
	if b.stream == nil {
		return PrepareResult{}, errors.New("YouTube bridge streaming runner is unavailable")
	}

	plan.URL = strings.TrimSpace(plan.URL)
	plan.Destination = strings.TrimSpace(plan.Destination)
	plan.StagingDirectory = strings.TrimSpace(plan.StagingDirectory)
	plan.CancelPath = strings.TrimSpace(plan.CancelPath)
	plan.Mode = strings.TrimSpace(plan.Mode)

	if plan.URL == "" {
		return PrepareResult{}, errors.New("YouTube prepare URL required")
	}
	if plan.Destination == "" {
		return PrepareResult{}, errors.New("YouTube prepare destination required")
	}
	if plan.StagingDirectory == "" {
		return PrepareResult{}, errors.New("YouTube prepare staging directory required")
	}
	if plan.Mode == "" {
		plan.Mode = "whole"
	}
	switch plan.Mode {
	case "whole", "split", "both":
	default:
		return PrepareResult{}, fmt.Errorf("unsupported YouTube prepare mode %q", plan.Mode)
	}
	if plan.CancelPath == "" {
		plan.CancelPath = filepath.Join(plan.StagingDirectory, ".cancel")
	}

	if err := os.MkdirAll(plan.StagingDirectory, 0o755); err != nil {
		return PrepareResult{}, fmt.Errorf("create import staging: %w", err)
	}
	planPath := filepath.Join(plan.StagingDirectory, "request.json")
	payload, err := json.Marshal(plan)
	if err != nil {
		return PrepareResult{}, fmt.Errorf("encode import request: %w", err)
	}
	if err := os.WriteFile(planPath, payload, 0o644); err != nil {
		return PrepareResult{}, fmt.Errorf("write import request: %w", err)
	}

	var (
		final           PrepareResult
		finalSeen       bool
		cancelled       bool
		providerFailure *Failure
		stderr          bytes.Buffer
	)

	emit := func(event PrepareEvent) {
		if onEvent != nil {
			onEvent(event)
		}
	}

	handleStdout := func(line []byte) {
		var envelope map[string]json.RawMessage
		if json.Unmarshal(line, &envelope) != nil {
			return
		}

		var eventType string
		_ = json.Unmarshal(envelope["type"], &eventType)
		switch eventType {
		case "progress":
			var row struct {
				Stage    string  `json:"stage"`
				Progress float64 `json:"progress"`
			}
			_ = json.Unmarshal(line, &row)
			emit(PrepareEvent{
				Type: "progress",
				Stage: strings.TrimSpace(row.Stage),
				Progress: int(row.Progress),
			})
			return
		case "cancelled":
			var row struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(line, &row)
			cancelled = true
			emit(PrepareEvent{
				Type: "cancelled",
				Message: strings.TrimSpace(row.Message),
			})
			return
		case "error":
			var row struct {
				Problem Problem `json:"problem"`
			}
			if json.Unmarshal(line, &row) == nil {
				if row.Problem.Stage == "" {
					row.Problem.Stage = "import"
				}
				if row.Problem.Provider == "" {
					row.Problem.Provider = "youtube"
				}
				providerFailure = &Failure{Problem: row.Problem}
				copy := row.Problem
				emit(PrepareEvent{Type: "error", Problem: &copy})
			}
			return
		}

		var row struct {
			OK      bool             `json:"ok"`
			Outputs []PreparedOutput `json:"outputs"`
			Staged  bool             `json:"staged"`
		}
		if json.Unmarshal(line, &row) == nil && row.OK {
			final = PrepareResult{
				Outputs: clonePreparedOutputs(row.Outputs),
				Staged: row.Staged,
			}
			finalSeen = true
		}
	}

	command := []string{b.Python, b.Script, "import", planPath}
	runErr := b.stream(
		ctx,
		command,
		handleStdout,
		func(line []byte) {
			stderr.Write(line)
			stderr.WriteByte('\n')
		},
	)

	switch {
	case cancelled:
		return PrepareResult{}, ErrPrepareCancelled
	case providerFailure != nil:
		return PrepareResult{}, providerFailure
	case runErr != nil:
		return PrepareResult{}, &Failure{
			Problem: parseProblem(stderr.Bytes(), "import"),
		}
	case !finalSeen || !final.Staged:
		return PrepareResult{}, errors.New("YouTube import preparation did not return staged output")
	default:
		return final, nil
	}
}

func (b *Bridge) runStreamingProcess(
	ctx context.Context,
	command []string,
	onStdout func([]byte),
	onStderr func([]byte),
) error {
	if len(command) < 2 {
		return errors.New("YouTube bridge streaming command is incomplete")
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	if len(b.Env) > 0 {
		cmd.Env = append([]string(nil), b.Env...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var (
		wg       sync.WaitGroup
		stderrErr error
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
		for scanner.Scan() {
			if onStderr != nil {
				onStderr(append([]byte(nil), scanner.Bytes()...))
			}
		}
		stderrErr = scanner.Err()
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		if onStdout != nil {
			onStdout(append([]byte(nil), scanner.Bytes()...))
		}
	}
	stdoutErr := scanner.Err()
	waitErr := cmd.Wait()
	wg.Wait()

	if stdoutErr != nil {
		return stdoutErr
	}
	if stderrErr != nil {
		return stderrErr
	}
	return waitErr
}

func clonePreparedOutputs(input []PreparedOutput) []PreparedOutput {
	out := make([]PreparedOutput, len(input))
	for index, value := range input {
		out[index] = value
		if value.CatalogEntry != nil {
			out[index].CatalogEntry = make(map[string]any, len(value.CatalogEntry))
			for key, item := range value.CatalogEntry {
				out[index].CatalogEntry[key] = item
			}
		}
	}
	return out
}
