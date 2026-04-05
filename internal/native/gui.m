#import "gui.h"
#import <Cocoa/Cocoa.h>

extern void shadowplayGuiOnStartBuffer(void);
extern void shadowplayGuiOnStopBuffer(void);
extern void shadowplayGuiOnSaveClip(void);
extern void shadowplayGuiOnOpenFolder(void);
extern void shadowplayGuiOnQuit(void);

@interface SPMenuDelegate : NSObject
@property (nonatomic, strong) NSStatusItem *statusItem;
@property (nonatomic, strong) NSMenuItem *startItem;
@property (nonatomic, strong) NSMenuItem *stopItem;
@property (nonatomic, strong) NSMenuItem *saveItem;
@end

static SPMenuDelegate *g_menuDelegate;

@implementation SPMenuDelegate

- (void)setupMenu {
    self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];

    NSImage *icon = [NSImage imageWithSystemSymbolName:@"record.circle"
                                 accessibilityDescription:@"ShadowPlay"];
    if (icon) {
        [icon setTemplate:YES];
        self.statusItem.button.image = icon;
    } else {
        self.statusItem.button.title = @"SP";
    }

    NSMenu *menu = [[NSMenu alloc] init];

    self.startItem = [[NSMenuItem alloc] initWithTitle:@"Start Buffer"
                                                action:@selector(onStart:)
                                         keyEquivalent:@""];
    self.startItem.target = self;
    [menu addItem:self.startItem];

    self.stopItem = [[NSMenuItem alloc] initWithTitle:@"Stop Buffer"
                                               action:@selector(onStop:)
                                        keyEquivalent:@""];
    self.stopItem.target = self;
    self.stopItem.enabled = NO;
    [menu addItem:self.stopItem];

    [menu addItem:[NSMenuItem separatorItem]];

    self.saveItem = [[NSMenuItem alloc] initWithTitle:@"Save Clip"
                                               action:@selector(onSave:)
                                        keyEquivalent:@""];
    self.saveItem.target = self;
    self.saveItem.enabled = NO;
    [menu addItem:self.saveItem];

    [menu addItem:[NSMenuItem separatorItem]];

    NSMenuItem *openItem = [[NSMenuItem alloc] initWithTitle:@"Open Clips Folder"
                                                      action:@selector(onOpenFolder:)
                                               keyEquivalent:@""];
    openItem.target = self;
    [menu addItem:openItem];

    [menu addItem:[NSMenuItem separatorItem]];

    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit"
                                                      action:@selector(onQuit:)
                                               keyEquivalent:@"q"];
    quitItem.target = self;
    [menu addItem:quitItem];

    self.statusItem.menu = menu;
}

- (void)setBuffering:(BOOL)active {
    self.startItem.enabled = !active;
    self.stopItem.enabled = active;
    self.saveItem.enabled = active;

    NSString *symbolName = active ? @"record.circle.fill" : @"record.circle";
    NSImage *icon = [NSImage imageWithSystemSymbolName:symbolName
                                 accessibilityDescription:@"ShadowPlay"];
    if (icon) {
        [icon setTemplate:YES];
        self.statusItem.button.image = icon;
    }
}

- (void)onStart:(id)sender {
    shadowplayGuiOnStartBuffer();
}

- (void)onStop:(id)sender {
    shadowplayGuiOnStopBuffer();
}

- (void)onSave:(id)sender {
    shadowplayGuiOnSaveClip();
}

- (void)onOpenFolder:(id)sender {
    shadowplayGuiOnOpenFolder();
}

- (void)onQuit:(id)sender {
    shadowplayGuiOnQuit();
}

@end

static void sp_gui_install_status_item_on_main(void) {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    g_menuDelegate = [[SPMenuDelegate alloc] init];
    [g_menuDelegate setupMenu];
    if (@available(macOS 11.0, *)) {
        g_menuDelegate.statusItem.visible = YES;
    }
}

void sp_gui_install_status_item_sync(void) {
    if ([NSThread isMainThread]) {
        sp_gui_install_status_item_on_main();
        return;
    }
    dispatch_sync(dispatch_get_main_queue(), ^{
        sp_gui_install_status_item_on_main();
    });
}

void sp_gui_set_buffering(int active) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [g_menuDelegate setBuffering:(active != 0)];
    });
}

void sp_gui_quit(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp terminate:nil];
    });
}
