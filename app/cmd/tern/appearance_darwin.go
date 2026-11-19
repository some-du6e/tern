package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>

void ternAppearanceChanged(int dark);

// systemDark reports whether macOS is in dark mode, as T3's
// prefers-color-scheme sees it. The defaults key is set while dark mode is
// on, including when Auto has switched to it.
static int systemDark(void) {
	@autoreleasepool {
		NSString *style = [[NSUserDefaults standardUserDefaults] stringForKey:@"AppleInterfaceStyle"];
		return style != nil && [style caseInsensitiveCompare:@"Dark"] == NSOrderedSame;
	}
}

// watchAppearance tells Go when the system switches, without polling.
static void watchAppearance(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[[NSDistributedNotificationCenter defaultCenter] addObserverForName:@"AppleInterfaceThemeChangedNotification"
			object:nil queue:[NSOperationQueue mainQueue] usingBlock:^(NSNotification *n) {
				ternAppearanceChanged(systemDark());
			}];
	});
}
*/
import "C"

var appearanceChanged func(dark bool)

// watchDark returns whether the system is in dark mode and calls changed
// whenever that changes.
func watchDark(changed func(dark bool)) bool {
	appearanceChanged = changed
	C.watchAppearance()
	return C.systemDark() != 0
}
