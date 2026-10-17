package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/vgong24/VexstreamMusic/v2/desktop/app"
	"github.com/vgong24/VexstreamMusic/v2/desktop/appdata"
	"github.com/vgong24/VexstreamMusic/v2/desktop/lifecycle"
)

const version = "4.0.0"

// buildRef may be supplied by release tooling. The executable digest is always appended so process
// reuse remains exact even when two builds accidentally receive the same human/source label.
var buildRef = ""

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "VexStream Desktop V4:", err)
		os.Exit(1)
	}
}

func run() error {
	layout := appdata.At(appdata.Base())
	ref, err := effectiveBuildRef()
	if err != nil {
		return err
	}

	if url, ok := lifecycle.Reusable(
		layout.CurrentURL,
		version,
		ref,
	); ok {
		return lifecycle.OpenBrowser(url)
	}

	// A stale V4 current-url file is replaceable, but a live non-matching V4 is
	// another writer. Exact reuse above is the only multi-launch shortcut.
	if claim, ok := lifecycle.CurrentLiveClaim(
		layout.CurrentURL,
	); ok {
		return differentV4BuildConflict(claim, ref)
	}

	if claim, ok := lifecycle.CurrentLiveClaim(
		layout.ProtectedCurrentURL,
	); ok {
		return protectedDesktopConflict(claim)
	}

	home, _ := os.UserHomeDir()
	application, err := app.Open(app.Options{
		Version:  version,
		BuildRef: ref,
		Layout:   layout,
		Home:     home,
	})
	if err != nil {
		return err
	}
	defer application.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()

	url := "http://" + listener.Addr().String() + "/"
	if err := lifecycle.Claim(layout.CurrentURL, url); err != nil {
		return err
	}
	defer func() {
		_ = lifecycle.ClearIfOwned(layout.CurrentURL, url)
	}()

	for _, warning := range application.StartupWarnings() {
		fmt.Fprintln(os.Stderr, "VexStream V4 warning:", warning)
	}

	server := &http.Server{
		Handler:           application.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	protectedConflict := make(chan error, 1)
	monitorDone := make(chan struct{})
	defer close(monitorDone)

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-monitorDone:
				return
			case <-ticker.C:
				claim, ok := lifecycle.CurrentLiveClaim(
					layout.ProtectedCurrentURL,
				)
				if !ok {
					continue
				}
				err := protectedDesktopConflict(claim)
				select {
				case protectedConflict <- err:
				default:
				}
				shutdown, cancel := context.WithTimeout(
					context.Background(),
					3*time.Second,
				)
				_ = server.Shutdown(shutdown)
				cancel()
				return
			}
		}
	}()
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		select {
		case <-application.ExitRequested():
		case <-ctx.Done():
		}
		shutdown, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()

	go func() {
		time.Sleep(250 * time.Millisecond)
		if err := lifecycle.OpenBrowser(url); err != nil {
			fmt.Fprintln(os.Stderr, "VexStream V4 browser:", err)
		}
	}()

	fmt.Println("VexStream Music", version)
	fmt.Println("Build:", ref)
	fmt.Println("Local:", url)

	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		select {
		case conflict := <-protectedConflict:
			return conflict
		default:
			return nil
		}
	}
	return err
}

func differentV4BuildConflict(
	claim lifecycle.LiveClaim,
	requestedBuildRef string,
) error {
	runningBuildRef := strings.TrimSpace(claim.Identity.BuildRef)
	if runningBuildRef == "" {
		runningBuildRef = "unknown-build"
	}
	requestedBuildRef = strings.TrimSpace(requestedBuildRef)
	if requestedBuildRef == "" {
		requestedBuildRef = "requested-build"
	}
	return fmt.Errorf(
		"VexStream 4.0 is already running at %s with build %s; "+
			"the requested build is %s. Exit the running V4 before starting "+
			"a different V4 build so two desktop processes never write the "+
			"same V4 and shared Library state at the same time",
		claim.URL,
		runningBuildRef,
		requestedBuildRef,
	)
}

func protectedDesktopConflict(
	claim lifecycle.LiveClaim,
) error {
	version := strings.TrimSpace(claim.Identity.Version)
	if version == "" {
		version = "another protected build"
	}
	return fmt.Errorf(
		"VexStream %s is already running at %s. "+
			"Exit the protected desktop before starting VexStream 4.0 so "+
			"both versions never write shared Library state at the same time",
		version,
		claim.URL,
	)
}

func effectiveBuildRef() (string, error) {
	label := strings.TrimSpace(buildRef)
	if label == "" {
		label = "vexstream.desktop-v4"
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable for build identity: %w", err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		return "", fmt.Errorf("read executable for build identity: %w", err)
	}
	digest := sha256.Sum256(data)
	return label + ".bin-" + hex.EncodeToString(digest[:8]), nil
}
