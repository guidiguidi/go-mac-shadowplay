#import "gui.h"
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

extern void shadowplayGuiOnStartBuffer(void);
extern void shadowplayGuiOnStopBuffer(void);
extern void shadowplayGuiOnSaveClip(void);
extern void shadowplayGuiOnOpenFolder(void);
extern void shadowplayGuiOnQuit(void);
extern void shadowplayGuiOnPreferences(void);

@interface SPMenuDelegate : NSObject
@property (nonatomic, strong) NSStatusItem *statusItem;
@property (nonatomic, strong) NSMenuItem *startItem;
@property (nonatomic, strong) NSMenuItem *stopItem;
@property (nonatomic, strong) NSMenuItem *saveItem;
@end

@interface SPPrefsActions : NSObject
@property (nonatomic, copy) NSArray<NSTextField *> *fields;
@property (nonatomic, copy) NSArray<NSString *> *keys;
@property (nonatomic, copy) void (^done)(int ok, char *jsonOrNull);
- (void)ok:(id)sender;
- (void)cancel:(id)sender;
@end

@implementation SPPrefsActions
- (void)ok:(id)sender {
    NSMutableDictionary *out = [NSMutableDictionary dictionary];
    for (NSUInteger i = 0; i < self.keys.count && i < self.fields.count; i++) {
        NSString *k = self.keys[i];
        NSString *s = self.fields[i].stringValue;
        if ([k isEqualToString:@"buffer_minutes"] || [k isEqualToString:@"clip_seconds"]) {
            out[k] = @([s intValue]);
        } else if ([k isEqualToString:@"segment_seconds"]) {
            out[k] = @([s doubleValue]);
        } else {
            out[k] = s ?: @"";
        }
    }
    NSError *err = nil;
    NSData *jd = [NSJSONSerialization dataWithJSONObject:out options:0 error:&err];
    if (!jd || err) {
        if (self.done) {
            self.done(0, NULL);
        }
        [NSApp stopModalWithCode:0];
        return;
    }
    NSString *js = [[NSString alloc] initWithData:jd encoding:NSUTF8StringEncoding];
    const char *utf = [js UTF8String];
    char *copy = utf ? strdup(utf) : NULL;
    if (self.done) {
        self.done(copy ? 1 : 0, copy);
    }
    [NSApp stopModalWithCode:copy ? 1 : 0];
}

- (void)cancel:(id)sender {
    if (self.done) {
        self.done(0, NULL);
    }
    [NSApp stopModalWithCode:0];
}
@end

static NSString *SPJsonStringForValue(id v) {
    if ([v isKindOfClass:[NSNumber class]]) {
        return [(NSNumber *)v stringValue];
    }
    if ([v isKindOfClass:[NSString class]]) {
        return (NSString *)v;
    }
    return @"";
}

static NSStackView *SPPrefsLabeledRow(NSString *labelText, NSString *value) {
    NSTextField *label = [[NSTextField alloc] initWithFrame:NSZeroRect];
    label.stringValue = labelText ?: @"";
    label.bezeled = NO;
    label.drawsBackground = NO;
    label.editable = NO;
    label.selectable = NO;
    label.alignment = NSTextAlignmentRight;
    [label setContentHuggingPriority:NSLayoutPriorityRequired
                      forOrientation:NSLayoutConstraintOrientationHorizontal];

    NSTextField *field = [[NSTextField alloc] initWithFrame:NSZeroRect];
    field.stringValue = value ?: @"";
    [field setContentHuggingPriority:NSLayoutPriorityDefaultLow
                    forOrientation:NSLayoutConstraintOrientationHorizontal];

    NSStackView *row = [[NSStackView alloc] init];
    row.orientation = NSUserInterfaceLayoutOrientationHorizontal;
    row.spacing = 10;
    row.alignment = NSLayoutAttributeCenterY;
    [row addArrangedSubview:label];
    [row addArrangedSubview:field];
    objc_setAssociatedObject(row, "field", field, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    return row;
}

static NSTextField *SPFieldFromRow(NSStackView *row) {
    return objc_getAssociatedObject(row, "field");
}

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

    NSMenuItem *prefsItem = [[NSMenuItem alloc] initWithTitle:@"Preferences…"
                                                       action:@selector(onPrefs:)
                                                keyEquivalent:@","];
    prefsItem.keyEquivalentModifierMask = NSEventModifierFlagCommand;
    prefsItem.target = self;
    [menu addItem:prefsItem];

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

- (void)onPrefs:(id)sender {
    shadowplayGuiOnPreferences();
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

int sp_gui_prefs_modal(const char *json_in, char **json_out) {
    if (json_out) {
        *json_out = NULL;
    }
    if (!json_in || !json_out) {
        return 0;
    }

    __block int ret = 0;
    __block char *outStr = NULL;

    dispatch_sync(dispatch_get_main_queue(), ^{
        NSData *data = [NSData dataWithBytes:json_in length:strlen(json_in)];
        NSError *err = nil;
        id obj = [NSJSONSerialization JSONObjectWithData:data options:0 error:&err];
        if (![obj isKindOfClass:[NSDictionary class]]) {
            return;
        }
        NSDictionary *dict = (NSDictionary *)obj;

        NSArray<NSString *> *keys = @[
            @"buffer_minutes", @"clip_seconds", @"segment_seconds", @"temp_dir", @"output_dir",
            @"save_hotkey", @"record_hotkey"
        ];
        NSArray<NSString *> *labels = @[
            @"Buffer (minutes):", @"Clip (seconds):", @"Segment (seconds):", @"Temp folder:",
            @"Clips folder:", @"Save hotkey:", @"Record hotkey:"
        ];

        NSMutableArray<NSTextField *> *fields = [NSMutableArray array];
        NSMutableArray<NSView *> *rows = [NSMutableArray array];
        for (NSUInteger i = 0; i < keys.count; i++) {
            NSString *val = SPJsonStringForValue(dict[keys[i]]);
            NSStackView *row = SPPrefsLabeledRow(labels[i], val);
            [rows addObject:row];
            [fields addObject:SPFieldFromRow(row)];
        }

        NSStackView *stack = [[NSStackView alloc] init];
        stack.orientation = NSUserInterfaceLayoutOrientationVertical;
        stack.spacing = 10;
        stack.alignment = NSLayoutAttributeLeading;
        for (NSView *v in rows) {
            [stack addArrangedSubview:v];
        }

        NSButton *okBtn = [NSButton buttonWithTitle:@"OK" target:nil action:nil];
        okBtn.bezelStyle = NSBezelStyleRounded;
        okBtn.keyEquivalent = @"\r";
        NSButton *cancelBtn = [NSButton buttonWithTitle:@"Cancel" target:nil action:nil];
        cancelBtn.bezelStyle = NSBezelStyleRounded;
        cancelBtn.keyEquivalent = @"\e";

        NSStackView *btnRow = [[NSStackView alloc] init];
        btnRow.orientation = NSUserInterfaceLayoutOrientationHorizontal;
        btnRow.spacing = 12;
        [btnRow addArrangedSubview:okBtn];
        [btnRow addArrangedSubview:cancelBtn];

        NSStackView *root = [[NSStackView alloc] init];
        root.orientation = NSUserInterfaceLayoutOrientationVertical;
        root.spacing = 16;
        root.edgeInsets = NSEdgeInsetsMake(20, 20, 20, 20);
        [root addArrangedSubview:stack];
        [root addArrangedSubview:btnRow];

        NSRect r = NSMakeRect(0, 0, 520, 420);
        NSPanel *panel = [[NSPanel alloc] initWithContentRect:r
                                                      styleMask:(NSWindowStyleMaskTitled | NSWindowStyleMaskClosable)
                                                        backing:NSBackingStoreBuffered
                                                          defer:NO];
        panel.title = @"ShadowPlay Settings";
        panel.contentView = root;
        [root setFrame:NSMakeRect(0, 0, 520, 420)];

        SPPrefsActions *acts = [[SPPrefsActions alloc] init];
        acts.fields = fields;
        acts.keys = keys;
        acts.done = ^(int ok, char *jsonOrNull) {
            if (ok && jsonOrNull) {
                ret = 1;
                outStr = jsonOrNull;
            }
        };
        okBtn.target = acts;
        okBtn.action = @selector(ok:);
        cancelBtn.target = acts;
        cancelBtn.action = @selector(cancel:);

        [panel center];
        NSInteger code = [NSApp runModalForWindow:panel];
        (void)code;
        [panel orderOut:nil];
    });

    *json_out = outStr;
    return ret;
}

void sp_gui_alert(const char *title, const char *message) {
    if (!title) {
        title = "";
    }
    if (!message) {
        message = "";
    }
    dispatch_sync(dispatch_get_main_queue(), ^{
        NSAlert *alert = [[NSAlert alloc] init];
        alert.messageText = [NSString stringWithUTF8String:title];
        alert.informativeText = [NSString stringWithUTF8String:message];
        alert.alertStyle = NSAlertStyleInformational;
        [alert runModal];
    });
}
