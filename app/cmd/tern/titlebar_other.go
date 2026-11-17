//go:build !darwin

package main

import (
	"gioui.org/app"
	"gioui.org/io/event"
)

// Elsewhere the window keeps its own title bar.
const titleInset = 0

var windowOptions []app.Option

func platformEvent(event.Event) {}
