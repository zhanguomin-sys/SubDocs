// Cocoa side of the Mac launcher: receives the launch, reopen and
// myautoapi:// events and passes them to Go.
#import <Cocoa/Cocoa.h>
#include "_cgo_export.h"

@interface UGDelegate : NSObject <NSApplicationDelegate>
@end

@implementation UGDelegate
// Registered here, before launch finishes, so the event that launched the
// app (oapp for a double click, GURL for a link) already reaches us.
- (void)applicationWillFinishLaunching:(NSNotification *)notification {
    NSAppleEventManager *manager = [NSAppleEventManager sharedAppleEventManager];
    [manager setEventHandler:self andSelector:@selector(handleURL:withReply:)
               forEventClass:kInternetEventClass andEventID:kAEGetURL];
    [manager setEventHandler:self andSelector:@selector(handleOpen:withReply:)
               forEventClass:kCoreEventClass andEventID:kAEOpenApplication];
    [manager setEventHandler:self andSelector:@selector(handleOpen:withReply:)
               forEventClass:kCoreEventClass andEventID:kAEReopenApplication];
}

- (void)handleURL:(NSAppleEventDescriptor *)event withReply:(NSAppleEventDescriptor *)reply {
    NSString *text = [[event paramDescriptorForKeyword:keyDirectObject] stringValue];
    goOpenURL((char *)[(text ?: @"") UTF8String]);
}

- (void)handleOpen:(NSAppleEventDescriptor *)event withReply:(NSAppleEventDescriptor *)reply {
    goOpenApplication();
}
@end

static UGDelegate *delegate;

void runApplication(void) {
    @autoreleasepool {
        NSApplication *application = [NSApplication sharedApplication];
        delegate = [[UGDelegate alloc] init];
        [application setDelegate:delegate];
        [application run];
    }
}

void stopApplication(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp terminate:nil];
    });
}
