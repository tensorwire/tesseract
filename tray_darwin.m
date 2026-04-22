#import <Cocoa/Cocoa.h>

extern void goTrayShow();
extern void goTrayQuit();

static NSStatusItem *statusItem = nil;

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
    dispatch_async(dispatch_get_main_queue(), ^{
        NSStatusBar *bar = [NSStatusBar systemStatusBar];
        statusItem = [bar statusItemWithLength:NSSquareStatusItemLength];

        NSData *data = [NSData dataWithBytes:iconData length:iconLen];
        NSImage *icon = [[NSImage alloc] initWithData:data];
        [icon setSize:NSMakeSize(18, 18)];
        [icon setTemplate:YES];
        statusItem.button.image = icon;

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
    });
}

void TrayRemove() {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem) {
            [[NSStatusBar systemStatusBar] removeStatusItem:statusItem];
            statusItem = nil;
        }
    });
}
