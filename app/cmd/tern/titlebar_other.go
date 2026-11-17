//go:build !darwin && !windows

package main

import (
	"gioui.org/app"
	"gioui.org/io/event"
)

// Elsewhere the window keeps its own title bar.
const titleInset = 0

var windowOptions []app.Option

// titleDark has no native title bar to match here.
func titleDark(bool) {}

func platformEvent(event.Event) {}
