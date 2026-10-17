//go:build !windows

package main

import "syscall"

func sysProcHidden() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}

func sysProcConsole() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
