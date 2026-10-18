// Package lifecycle owns V4 desktop process identity and current-server placement.
package lifecycle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Identity struct {
	OK       bool   `json:"ok"`
	Version  string `json:"version"`
	BuildRef string `json:"buildRef"`
}

type LiveClaim struct {
	URL      string
	Identity Identity
}

func Probe(url string) Identity {
	url = strings.TrimSpace(url)
	if url == "" {
		return Identity{}
	}
	client := http.Client{Timeout: 700 * time.Millisecond}
	response, err := client.Get(strings.TrimRight(url, "/") + "/health")
	if err != nil {
		return Identity{}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Identity{}
	}
	var identity Identity
	if json.NewDecoder(response.Body).Decode(&identity) != nil ||
		!identity.OK {
		return Identity{}
	}
	return identity
}

func RequestExit(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return errors.New("VexStream exit URL required")
	}

	request, err := http.NewRequest(
		http.MethodPost,
		strings.TrimRight(url, "/")+"/api/system/exit",
		bytes.NewBufferString(`{"confirm":"EXIT_VEXSTREAM"}`),
	)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request VexStream exit: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"VexStream exit returned HTTP %d",
			response.StatusCode,
		)
	}
	return nil
}

func WaitUntilUnavailable(
	url string,
	timeout time.Duration,
) bool {
	url = strings.TrimSpace(url)
	if url == "" {
		return true
	}
	if timeout <= 0 {
		return !Probe(url).OK
	}

	deadline := time.Now().Add(timeout)
	for {
		if !Probe(url).OK {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func WaitUntilClaimReleased(
	currentURLPath string,
	url string,
	timeout time.Duration,
) bool {
	currentURLPath = strings.TrimSpace(currentURLPath)
	url = strings.TrimSpace(url)
	if currentURLPath == "" || url == "" {
		return false
	}

	released := func() bool {
		data, err := os.ReadFile(currentURLPath)
		if os.IsNotExist(err) {
			return true
		}
		if err != nil {
			return false
		}
		return strings.TrimSpace(string(data)) != url
	}

	if timeout <= 0 {
		return released()
	}

	deadline := time.Now().Add(timeout)
	for {
		if released() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func CurrentLiveClaim(currentURLPath string) (LiveClaim, bool) {
	data, err := os.ReadFile(currentURLPath)
	if err != nil {
		return LiveClaim{}, false
	}
	url := strings.TrimSpace(string(data))
	if url == "" {
		return LiveClaim{}, false
	}
	identity := Probe(url)
	if !identity.OK {
		return LiveClaim{}, false
	}
	return LiveClaim{
		URL:      url,
		Identity: identity,
	}, true
}

func Reusable(
	currentURLPath string,
	version string,
	buildRef string,
) (string, bool) {
	data, err := os.ReadFile(currentURLPath)
	if err != nil {
		return "", false
	}
	url := strings.TrimSpace(string(data))
	if url == "" {
		return "", false
	}
	identity := Probe(url)
	return url,
		identity.OK &&
			identity.Version == version &&
			identity.BuildRef == buildRef
}

func Claim(currentURLPath, url string) error {
	currentURLPath = strings.TrimSpace(currentURLPath)
	url = strings.TrimSpace(url)
	if currentURLPath == "" || url == "" {
		return errors.New("current URL path and URL required")
	}
	if err := os.MkdirAll(filepath.Dir(currentURLPath), 0o755); err != nil {
		return err
	}
	tmp := currentURLPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(url+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, currentURLPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func ClearIfOwned(currentURLPath, url string) error {
	data, err := os.ReadFile(currentURLPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(data)) != strings.TrimSpace(url) {
		return nil
	}
	if err := os.Remove(currentURLPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
