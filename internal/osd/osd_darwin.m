#import <Cocoa/Cocoa.h>
#import <objc/message.h>
#include "osd_darwin.h"

// The system indicator has no public interface. Before macOS 26 the private
// OSDManager class draws it. From macOS 26 that class draws a new indicator
// that shows an external monitor as locked, so monctl asks the helper service
// behind it directly, which still draws the usable one.
static const long long monctlBrightnessImage = 1;
static const unsigned int monctlPriority = 0x1f4;
static const unsigned int monctlFadeMsec = 1000;

@protocol MonctlOSDHelper
- (void)showImage:(long long)image onDisplayID:(unsigned int)display priority:(unsigned int)priority msecUntilFade:(unsigned int)fade filledChiclets:(unsigned int)filled totalChiclets:(unsigned int)total locked:(BOOL)locked;
@end

static NSXPCConnection *monctlConnection;

static id<MonctlOSDHelper> monctlHelper(void) {
	if (monctlConnection == nil) {
		NSXPCConnection *connection = [[NSXPCConnection alloc] initWithMachServiceName:@"com.apple.OSDUIHelper" options:0];
		connection.remoteObjectInterface = [NSXPCInterface interfaceWithProtocol:@protocol(MonctlOSDHelper)];
		connection.invalidationHandler = ^{
			dispatch_async(dispatch_get_main_queue(), ^{ monctlConnection = nil; });
		};
		[connection resume];
		monctlConnection = connection;
	}
	return [monctlConnection remoteObjectProxyWithErrorHandler:^(NSError *error) {
		dispatch_async(dispatch_get_main_queue(), ^{
			[monctlConnection invalidate];
			monctlConnection = nil;
		});
	}];
}

static id<MonctlOSDHelper> monctlManager(void) {
	static id manager;
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		[[NSBundle bundleWithPath:@"/System/Library/PrivateFrameworks/OSD.framework"] load];
		Class class = NSClassFromString(@"OSDManager");
		SEL shared = NSSelectorFromString(@"sharedManager");
		if (class != Nil && [class respondsToSelector:shared]) {
			id found = ((id (*)(id, SEL))objc_msgSend)(class, shared);
			if ([found respondsToSelector:@selector(showImage:onDisplayID:priority:msecUntilFade:filledChiclets:totalChiclets:locked:)]) {
				manager = found;
			}
		}
	});
	return manager;
}

static BOOL monctlUsesHelper(void) {
	if (@available(macOS 26.0, *)) {
		return YES;
	}
	return NO;
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
	return monctlUsesHelper() || monctlManager() != nil;
}

void monctlOSDShow(int filled, int total) {
	dispatch_async(dispatch_get_main_queue(), ^{
		id<MonctlOSDHelper> target = monctlUsesHelper() ? monctlHelper() : monctlManager();
		[target showImage:monctlBrightnessImage onDisplayID:monctlExternalDisplay() priority:monctlPriority
			msecUntilFade:monctlFadeMsec filledChiclets:(unsigned int)filled totalChiclets:(unsigned int)total locked:NO];
	});
}
