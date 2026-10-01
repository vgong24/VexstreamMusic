//go:build windows

package main

import "syscall"

func sysProcHidden() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}

func sysProcConsole() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000010}
}
