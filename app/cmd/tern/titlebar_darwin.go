package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdint.h>
#import <AppKit/AppKit.h>

// The traffic lights where T3's desktop app puts them: Electron's
// trafficLightPosition {x: 16, y: 19}, in a 52pt top bar (19 = 52/2 - 7).
// The title bar's container is made the bar's height and the buttons moved
// into it, keeping their native spacing. AppKit lays them out again on
// resizes and full screen, so they're moved back each time.
static const CGFloat barH = 52, lightsX = 16, lightsTop = 19;

static void placeLights(NSWindow *w) {
	if (w.styleMask & NSWindowStyleMaskFullScreen) {
		return;
	}
	NSButton *b[3] = {
		[w standardWindowButton:NSWindowCloseButton],
		[w standardWindowButton:NSWindowMiniaturizeButton],
		[w standardWindowButton:NSWindowZoomButton],
	};
	if (b[0] == nil || b[0].superview == nil) {
		return;
	}
	NSView *bar = b[0].superview.superview;
	NSRect f = bar.frame;
	f.size.height = barH;
	f.origin.y = w.frame.size.height - barH;
	bar.frame = f;
	// Shift them as a group, so they keep the system's spacing. The
	// container isn't flipped: y counts up from the bar's bottom.
	CGFloat base = b[0].frame.origin.x;
	for (int i = 0; i < 3; i++) {
		if (b[i] == nil) {
			continue;
		}
		b[i].hidden = NO;
		[b[i] setFrameOrigin:NSMakePoint(lightsX + (b[i].frame.origin.x - base), barH - lightsTop - b[i].frame.size.height)];
	}
}

static void showLights(uintptr_t viewRef) {
	NSView *view = (NSView *)(void *)viewRef;
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *w = view.window;
		if (w == nil) {
			return;
		}
		static BOOL watching = NO;
		if (!watching) {
			watching = YES;
			NSNotificationName names[] = {
				NSWindowDidResizeNotification, NSWindowDidExitFullScreenNotification,
				NSWindowDidBecomeKeyNotification, NSWindowDidResignKeyNotification,
				NSWindowDidChangeBackingPropertiesNotification,
			};
			for (int i = 0; i < 5; i++) {
				[[NSNotificationCenter defaultCenter] addObserverForName:names[i] object:w
					queue:[NSOperationQueue mainQueue] usingBlock:^(NSNotification *n) { placeLights(w); }];
			}
		}
		placeLights(w);
	});
}
*/
import "C"

import (
	"gioui.org/app"
	"gioui.org/io/event"
)

// The traffic lights take T3's 90pt at the top bar's left, so the sidebar
// trigger starts after them, and tern draws the title bar itself.
const (
	controlsLeft = 90
	frameless    = true
)

var windowOptions = []app.Option{app.Decorated(false)}

// titleDark has nothing to do on a Mac: there is no title bar, and the
// traffic lights follow the system.
func titleDark(bool) {}

var view uintptr

// platformEvent puts the traffic lights back once the window has a view,
// and after every change to its options, which hides them again.
func platformEvent(e event.Event) {
	switch e := e.(type) {
	case app.AppKitViewEvent:
		view = e.View
	case app.ConfigEvent:
	default:
		return
	}
	if view != 0 {
		C.showLights(C.uintptr_t(view))
	}
}
