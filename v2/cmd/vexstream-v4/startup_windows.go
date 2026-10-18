//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/vgong24/VexstreamMusic/v2/desktop/lifecycle"
)

const (
	messageBoxOK            = 0x00000000
	messageBoxYesNo         = 0x00000004
	messageBoxIconError     = 0x00000010
	messageBoxIconQuestion  = 0x00000020
	messageBoxSetForeground = 0x00010000

	messageBoxResultYes = 6
	messageBoxResultNo  = 7
)

var (
	user32DLL       = syscall.NewLazyDLL("user32.dll")
	messageBoxWProc = user32DLL.NewProc("MessageBoxW")
)

func handoffProtectedDesktop(
	currentURLPath string,
	claim lifecycle.LiveClaim,
) (bool, error) {
	version := strings.TrimSpace(claim.Identity.Version)
	if version == "" {
		version = "an older VexStream"
	}

	message :=
		"VexStream " + version + " is already running and owns your existing music library.\r\n\r\n" +
			"VexStream 4.0 cannot safely write the same library at the same time.\r\n\r\n" +
			"Exit the running VexStream cleanly and start V4 now?\r\n\r\n" +
			"Active playback or imports in the running app will stop. " +
			"Choose No to keep it running and reopen its browser."

	choice, err := windowsMessageBox(
		message,
		"VexStream Music 4.0",
		messageBoxYesNo|
			messageBoxIconQuestion|
			messageBoxSetForeground,
	)
	if err != nil {
		return false, fmt.Errorf(
			"show protected VexStream handoff: %w",
			err,
		)
	}

	switch choice {
	case messageBoxResultNo:
		if err := lifecycle.OpenBrowser(claim.URL); err != nil {
			return false, fmt.Errorf(
				"keep VexStream %s running, but could not reopen its browser: %w",
				version,
				err,
			)
		}
		return false, nil

	case messageBoxResultYes:
		if err := lifecycle.RequestExit(claim.URL); err != nil {
			return false, fmt.Errorf(
				"ask VexStream %s to exit cleanly: %w",
				version,
				err,
			)
		}
		if !lifecycle.WaitUntilUnavailable(
			claim.URL,
			5*time.Second,
		) {
			return false, fmt.Errorf(
				"VexStream %s did not stop its local server after the exit request",
				version,
			)
		}
		if !lifecycle.WaitUntilClaimReleased(
			currentURLPath,
			claim.URL,
			5*time.Second,
		) {
			return false, fmt.Errorf(
				"VexStream %s stopped serving but did not release ownership of the shared library",
				version,
			)
		}
		return true, nil

	default:
		return false, fmt.Errorf(
			"unexpected Windows handoff response: %d",
			choice,
		)
	}
}

func reportStartupFailure(
	startupErr error,
) {
	message :=
		"VexStream Music 4.0 could not start.\r\n\r\n" +
			startupErr.Error()

	if _, err := windowsMessageBox(
		message,
		"VexStream Music 4.0",
		messageBoxOK|
			messageBoxIconError|
			messageBoxSetForeground,
	); err != nil {
		fmt.Fprintln(
			os.Stderr,
			"VexStream Desktop V4:",
			startupErr,
			"(startup dialog failed:",
			err,
			")",
		)
		return
	}

	fmt.Fprintln(
		os.Stderr,
		"VexStream Desktop V4:",
		startupErr,
	)
}

func windowsMessageBox(
	message string,
	title string,
	flags uintptr,
) (uintptr, error) {
	messagePointer, err :=
		syscall.UTF16PtrFromString(
			message,
		)
	if err != nil {
		return 0, err
	}
	titlePointer, err :=
		syscall.UTF16PtrFromString(
			title,
		)
	if err != nil {
		return 0, err
	}

	result, _, callErr :=
		messageBoxWProc.Call(
			0,
			uintptr(
				unsafe.Pointer(
					messagePointer,
				),
			),
			uintptr(
				unsafe.Pointer(
					titlePointer,
				),
			),
			flags,
		)
	if result != 0 {
		return result, nil
	}

	if errno, ok := callErr.(syscall.Errno); ok &&
		errno != 0 {
		return 0, errno
	}
	return 0, errors.New(
		"Windows MessageBoxW returned no result",
	)
}
