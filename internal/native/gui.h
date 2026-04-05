#ifndef SHADOWPLAY_GUI_H
#define SHADOWPLAY_GUI_H

#ifdef __cplusplus
extern "C" {
#endif

void sp_gui_install_status_item_sync(void);

void sp_gui_set_buffering(int active);
void sp_gui_quit(void);

/* Returns 1 if user clicked OK (json_out is malloc'd UTF-8 JSON); 0 if cancelled. */
int sp_gui_prefs_modal(const char *json_in, char **json_out);

void sp_gui_alert(const char *title, const char *message);

#ifdef __cplusplus
}
#endif

#endif
