//go:build !windows

package youtube

import "syscall"

func hiddenProcAttr() *syscall.SysProcAttr {
	return nil
}
