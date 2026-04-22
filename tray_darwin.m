#import <Cocoa/Cocoa.h>

extern void goTrayShow(void);
extern void goTrayQuit(void);

static NSStatusItem * statusItem = nil;

@interface TrayDelegate : NSObject
- (void)showWindow:(id)sender;
- (void)quitApp:(id)sender;
@end

@implementation TrayDelegate
- (void)showWindow:(id)sender { goTrayShow(); }
- (void)quitApp:(id)sender { goTrayQuit(); }
@end

static TrayDelegate *trayDelegate = nil;

void TrayCreate(const void *iconData, int iconLen) {
    if (![NSThread isMainThread]) {
        dispatch_sync(dispatch_get_main_queue(), ^{
            TrayCreate(iconData, iconLen);
        });
        return;
    }

    NSLog(@"[tesseract] creating tray on main thread");

    statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
    [statusItem retain];

    NSStatusBarButton *button = statusItem.button;
    if (!button) {
        NSLog(@"[tesseract] ERROR: statusItem.button is nil");
        return;
    }

    NSData *data = [NSData dataWithBytes:iconData length:iconLen];
    NSImage *icon = [[NSImage alloc] initWithData:data];
    if (icon) {
        [icon setSize:NSMakeSize(18, 18)];
        [icon setTemplate:YES];
        button.image = icon;
    } else {
        button.title = @"T";
    }

    trayDelegate = [[TrayDelegate alloc] init];
    NSMenu *menu = [[NSMenu alloc] init];

    NSMenuItem *showItem = [[NSMenuItem alloc] initWithTitle:@"Show Tesseract"
        action:@selector(showWindow:) keyEquivalent:@""];
    [showItem setTarget:trayDelegate];
    [menu addItem:showItem];

    [menu addItem:[NSMenuItem separatorItem]];

    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit"
        action:@selector(quitApp:) keyEquivalent:@"q"];
    [quitItem setTarget:trayDelegate];
    [menu addItem:quitItem];

    statusItem.menu = menu;

    NSLog(@"[tesseract] tray created: button=%p menu=%p visible=%d",
          button, menu, statusItem.visible);
}

void TrayRemove(void) {
    if (statusItem) {
        [[NSStatusBar systemStatusBar] removeStatusItem:statusItem];
        [statusItem release];
        statusItem = nil;
    }
}
