package clip

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
#include <string.h>
#import <AppKit/AppKit.h>

enum { clipNone, clipText, clipPNG, clipFiles };

static void *copyOut(NSData *d, int *n) {
	*n = (int)d.length;
	void *p = malloc(d.length);
	memcpy(p, d.bytes, d.length);
	return p;
}

// readClip looks at the pasteboard the way read does on Windows: a PNG, then
// files, then text, then any other bitmap. Files come back as paths, one
// per line.
static int readClip(void **data, int *n) {
	@autoreleasepool {
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		NSData *d = [pb dataForType:NSPasteboardTypePNG];
		if (d.length > 0) {
			*data = copyOut(d, n);
			return clipPNG;
		}
		NSArray<NSURL *> *urls = [pb readObjectsForClasses:@[[NSURL class]]
			options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];
		if (urls.count > 0) {
			NSMutableArray *paths = [NSMutableArray array];
			for (NSURL *u in urls) {
				if (u.path) [paths addObject:u.path];
			}
			*data = copyOut([[paths componentsJoinedByString:@"\n"] dataUsingEncoding:NSUTF8StringEncoding], n);
			return clipFiles;
		}
		if ([pb stringForType:NSPasteboardTypeString] != nil) {
			return clipText;
		}
		d = [pb dataForType:NSPasteboardTypeTIFF];
		if (d.length > 0) {
			NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:d];
			NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
			if (png.length > 0) {
				*data = copyOut(png, n);
				return clipPNG;
			}
		}
		return clipNone;
	}
}
*/
import "C"

import (
	"strings"
	"unsafe"
)

func read() ([]Image, error) {
	var p unsafe.Pointer
	var n C.int
	kind := C.readClip(&p, &n)
	if p == nil {
		return nil, ErrNone
	}
	b := C.GoBytes(p, n)
	C.free(p)
	switch kind {
	case C.clipPNG:
		return fromPNG(b)
	case C.clipFiles:
		imgs, err := fromFiles(strings.Split(string(b), "\n"))
		if err != ErrNone {
			return imgs, err
		}
		// Files that aren't pictures paste as their names.
	}
	return nil, ErrNone
}
