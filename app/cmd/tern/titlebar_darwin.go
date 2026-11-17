package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdint.h>
#import <AppKit/AppKit.h>

// The traffic lights, centred in tern's 48pt top row instead of the 28pt
// title bar the window doesn't have: the title bar's container is made the
// row's height and the buttons moved into it, what Electron's
// trafficLightPosition does. AppKit lays them out again on resizes and
// full screen, so they're moved back each time.
static const CGFloat barH = 48, lightsX = 18, lightsGap = 20;

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
	for (int i = 0; i < 3; i++) {
		if (b[i] == nil) {
			continue;
		}
		b[i].hidden = NO;
		[b[i] setFrameOrigin:NSMakePoint(lightsX + i*lightsGap, (barH - b[i].frame.size.height) / 2)];
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

// titleInset is the room the traffic lights take at the sidebar's top left.
const titleInset = 84

var windowOptions = []app.Option{app.Decorated(false)}

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
