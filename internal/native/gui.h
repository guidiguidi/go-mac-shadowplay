#ifndef SHADOWPLAY_GUI_H
#define SHADOWPLAY_GUI_H

#ifdef __cplusplus
extern "C" {
#endif

/* Safe from any thread; blocks until the status item exists on the main thread. */
void sp_gui_install_status_item_sync(void);

void sp_gui_set_buffering(int active);
void sp_gui_quit(void);

#ifdef __cplusplus
}
#endif

#endif
