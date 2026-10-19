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

func handoffDifferentV4(
	currentURLPath string,
	claim lifecycle.LiveClaim,
	_ string,
) (bool, error) {
	message :=
		"Another VexStream 4.0 build is already running.\r\n\r\n" +
			"The VexStream you just prepared is a different build. " +
			"Only one VexStream can safely use your shared music-library state at a time.\r\n\r\n" +
			"Switch to the newly prepared VexStream now?\r\n\r\n" +
			"Yes — exit the running VexStream cleanly and continue.\r\n" +
			"No — keep the current VexStream running and reopen its browser.\r\n\r\n" +
			"Your music files are not deleted or moved by this switch."

	choice, err := windowsMessageBox(
		message,
		"VexStream Music 4.0 — Update ready",
		messageBoxYesNo|
			messageBoxIconQuestion|
			messageBoxSetForeground,
	)
	if err != nil {
		return false, fmt.Errorf(
			"show VexStream 4.0 update handoff: %w",
			err,
		)
	}

	switch choice {
	case messageBoxResultNo:
		if err := lifecycle.OpenBrowser(claim.URL); err != nil {
			return false, fmt.Errorf(
				"keep the running VexStream 4.0, but could not reopen its browser: %w",
				err,
			)
		}
		return false, nil

	case messageBoxResultYes:
		if err := lifecycle.RequestExit(claim.URL); err != nil {
			return false, fmt.Errorf(
				"ask the running VexStream 4.0 to exit cleanly: %w",
				err,
			)
		}
		if !lifecycle.WaitUntilUnavailable(
			claim.URL,
			5*time.Second,
		) {
			return false, errors.New(
				"the running VexStream 4.0 did not stop after the update handoff request",
			)
		}
		if err := completeWindowsOwnershipHandoff(
			currentURLPath,
			claim,
			"the running VexStream 4.0",
		); err != nil {
			return false, err
		}
		return true, nil

	default:
		return false, fmt.Errorf(
			"unexpected Windows update handoff response: %d",
			choice,
		)
	}
}

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
		if err := completeWindowsOwnershipHandoff(
			currentURLPath,
			claim,
			"VexStream "+version,
		); err != nil {
			return false, err
		}
		return true, nil

	default:
		return false, fmt.Errorf(
			"unexpected Windows handoff response: %d",
			choice,
		)
	}
}

func completeWindowsOwnershipHandoff(
	currentURLPath string,
	claim lifecycle.LiveClaim,
	ownerLabel string,
) error {
	// Current V4 builds retain their claim until feature-owned background
	// services have actually closed. Give them a bounded clean shutdown window.
	if lifecycle.WaitUntilClaimReleased(
		currentURLPath,
		claim.URL,
		10*time.Second,
	) {
		if next, ok := lifecycle.CurrentLiveClaim(currentURLPath); ok {
			return fmt.Errorf(
				"another VexStream became active at %s while %s was handing off ownership",
				next.URL,
				ownerLabel,
			)
		}
		return nil
	}

	// Builds predating PID-bearing health used the older cleanup order. If
	// such a build has definitely stopped serving but left only its exact URL
	// claim behind, recover that stale pointer rather than forcing the human
	// to hunt processes or delete state by hand.
	if claim.Identity.PID == 0 {
		if err := lifecycle.RecoverStoppedClaim(
			currentURLPath,
			claim.URL,
		); err != nil {
			return fmt.Errorf(
				"%s stopped serving, but its legacy ownership claim could not be recovered: %w",
				ownerLabel,
				err,
			)
		}
		if !lifecycle.WaitUntilClaimReleased(
			currentURLPath,
			claim.URL,
			time.Second,
		) {
			return fmt.Errorf(
				"%s stopped serving but its recovered ownership claim is still present",
				ownerLabel,
			)
		}
		if next, ok := lifecycle.CurrentLiveClaim(currentURLPath); ok {
			return fmt.Errorf(
				"another VexStream became active at %s during the ownership handoff",
				next.URL,
			)
		}
		return nil
	}

	return fmt.Errorf(
		"%s stopped serving but process %d has not finished releasing shared-library ownership",
		ownerLabel,
		claim.Identity.PID,
	)
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
