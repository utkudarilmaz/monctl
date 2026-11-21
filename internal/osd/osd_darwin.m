#import <Cocoa/Cocoa.h>
#import <objc/message.h>
#include "osd_darwin.h"

// The system indicator has no public interface. This uses the same private
// class that other brightness tools use, and does nothing when it is missing.
static const long long monctlBrightnessImage = 1;
static const unsigned int monctlPriority = 0x1f4;
static const unsigned int monctlFadeMsec = 1000;

static SEL monctlShowSelector(void) {
	return NSSelectorFromString(@"showImage:onDisplayID:priority:msecUntilFade:filledChiclets:totalChiclets:locked:");
}

static id monctlManager(void) {
	static id manager;
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		[[NSBundle bundleWithPath:@"/System/Library/PrivateFrameworks/OSD.framework"] load];
		Class class = NSClassFromString(@"OSDManager");
		SEL shared = NSSelectorFromString(@"sharedManager");
		if (class != Nil && [class respondsToSelector:shared]) {
			id found = ((id (*)(id, SEL))objc_msgSend)(class, shared);
			if ([found respondsToSelector:monctlShowSelector()]) {
				manager = found;
			}
		}
	});
	return manager;
}

static CGDirectDisplayID monctlExternalDisplay(void) {
	CGDirectDisplayID displays[16];
	uint32_t count = 0;
	if (CGGetOnlineDisplayList(16, displays, &count) == kCGErrorSuccess) {
		for (uint32_t i = 0; i < count; i++) {
			if (!CGDisplayIsBuiltin(displays[i])) {
				return displays[i];
			}
		}
	}
	return CGMainDisplayID();
}

int monctlOSDAvailable(void) {
	return monctlManager() != nil;
}

void monctlOSDShow(int filled, int total) {
	dispatch_async(dispatch_get_main_queue(), ^{
		id manager = monctlManager();
		if (manager == nil) {
			return;
		}
		((void (*)(id, SEL, long long, unsigned int, unsigned int, unsigned int, unsigned int, unsigned int, BOOL))objc_msgSend)(
			manager, monctlShowSelector(), monctlBrightnessImage, monctlExternalDisplay(),
			monctlPriority, monctlFadeMsec, (unsigned int)filled, (unsigned int)total, NO);
	});
}
