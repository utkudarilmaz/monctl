#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#import <IOKit/hidsystem/ev_keymap.h>
#include "keys_darwin.h"
#include "_cgo_export.h"

static CFMachPortRef monctlTap;

static CGEventRef monctlOnEvent(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *info) {
	if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
		CGEventTapEnable(monctlTap, true);
		return event;
	}
	if (type != NX_SYSDEFINED) {
		return event;
	}
	NSEvent *e = [NSEvent eventWithCGEvent:event];
	if (e == nil || e.subtype != NX_SUBTYPE_AUX_CONTROL_BUTTONS) {
		return event;
	}
	int key = (int)((e.data1 & 0xFFFF0000) >> 16);
	int down = ((e.data1 & 0xFF00) >> 8) == 0x0A;
	if (key != NX_KEYTYPE_BRIGHTNESS_UP && key != NX_KEYTYPE_BRIGHTNESS_DOWN) {
		return event;
	}
	return monctlBrightnessKey(key == NX_KEYTYPE_BRIGHTNESS_UP ? 1 : -1, down) ? NULL : event;
}

// Returns 0 when the tap runs, 1 without the Accessibility permission, 2 when
// the tap cannot be made.
int monctlStartTap(void) {
	NSDictionary *options = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};
	if (!AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options)) {
		return 1;
	}
	monctlTap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap, kCGEventTapOptionDefault,
		CGEventMaskBit(NX_SYSDEFINED), monctlOnEvent, NULL);
	if (monctlTap == NULL) {
		return 2;
	}
	CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, monctlTap, 0);
	CFRunLoopAddSource(CFRunLoopGetMain(), source, kCFRunLoopCommonModes);
	CFRelease(source);
	CGEventTapEnable(monctlTap, true);
	return 0;
}
