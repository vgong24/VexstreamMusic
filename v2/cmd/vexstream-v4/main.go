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

	home, _ := os.UserHomeDir()
	application, err := app.Open(app.Options{
		Version: version,
		BuildRef: ref,
		Layout: layout,
		Home: home,
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
		Handler: application.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
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
		return nil
	}
	return err
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
