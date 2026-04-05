#ifndef SHADOWPLAY_CAPTURE_H
#define SHADOWPLAY_CAPTURE_H

#ifdef __cplusplus
extern "C" {
#endif

int sp_capture_start(const char *output_path);
int sp_capture_stop(void);
int sp_capture_is_recording(void);

int sp_rolling_start(const char *output_dir, double segment_seconds, double max_buffer_seconds);
int sp_rolling_stop(void);
int sp_rolling_is_active(void);
int sp_rolling_export_last(const char *output_path, double duration_seconds);

#ifdef __cplusplus
}
#endif

#endif
