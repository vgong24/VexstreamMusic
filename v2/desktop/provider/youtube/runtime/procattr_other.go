//go:build !windows

package runtimeenv

import "syscall"

func hiddenProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
