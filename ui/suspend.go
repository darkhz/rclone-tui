//go:build !windows
// +build !windows

package ui

import (
	"syscall"

	"github.com/gdamore/tcell/v2"
)

// SuspendApp suspends the application.
func SuspendApp(t tcell.Screen) {
	t.Suspend()
	syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	t.Resume()
}
