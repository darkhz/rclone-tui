//go:build windows
// +build windows

package ui

import "github.com/gdamore/tcell/v2"

// SuspendApp is disabled in Windows.
func SuspendApp(t tcell.Screen) {
}
