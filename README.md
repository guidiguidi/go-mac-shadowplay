# go-mac-shadowplay

ShadowPlay-style screen capture for **macOS**, written in **Go** with a small Objective-C layer on top of [ScreenCaptureKit](https://developer.apple.com/documentation/screencapturekit) and AVFoundation (H.264 in MP4/MOV).

## Requirements

- macOS 12.3+ (ScreenCaptureKit)
- Go 1.22+
- Xcode Command Line Tools (`xcode-select --install`)

## Permissions

The first run opens **Screen Recording** permission for Terminal (or your IDE). Enable it under  
**System Settings → Privacy & Security → Screen Recording**.

Some global shortcuts may require **Accessibility** for the host app — grant if macOS prompts you.

## Build

```bash
make build
# or
CGO_ENABLED=1 go build -o shadowplay ./cmd/shadowplay
```

## Usage

**Manual recording** — stop with **Ctrl+C** or **`record_hotkey`** from config (default `cmd+shift+r`):

```bash
./shadowplay record -o ~/Movies/capture.mov
./shadowplay record -o ~/Movies/capture.mov -config ./shadowplay.example.yaml
```

**Instant replay buffer (CLI)** — keeps rolling segment files under `temp_dir`, trims by `buffer_minutes`:

```bash
./shadowplay buffer
./shadowplay buffer -config ./shadowplay.example.yaml
```

While buffer mode is running:

- **`save_hotkey`** (default `cmd+shift+s`) — export the last `clip_seconds` to `output_dir` as `clip_YYYYMMDD_HHMMSS.mp4`
- **`record_hotkey`** (default `cmd+shift+r`) — if different from `save_hotkey`, same save action (second shortcut)
- **Ctrl+C** — stop and exit

**Menu bar app** — native macOS status bar icon with Start/Stop/Save/Quit:

```bash
./shadowplay gui
./shadowplay gui -config ./shadowplay.example.yaml
```

The icon appears in the menu bar (SF Symbol `record.circle`). Click it to start/stop the buffer, save a clip, or open the clips folder. Global hotkeys from the config are also active while the buffer is running.

## Configuration

Optional YAML fields (see `shadowplay.example.yaml`):

| Field | Meaning |
|-------|---------|
| `buffer_minutes` | Approximate rolling history kept on disk |
| `clip_seconds` | Length of the file written when you press ⌘⇧S |
| `segment_seconds` | Length of each internal segment file |
| `temp_dir` | Where segment `.mov` files are stored |
| `output_dir` | Where exported clips are saved |
| `save_hotkey` | Global shortcut to save a replay clip in buffer mode |
| `record_hotkey` | In **record** mode: stop recording. In **buffer** mode: extra save shortcut if different from `save_hotkey` |

Hotkey format: modifiers and key separated by `+`, lowercase. Modifiers: `cmd`, `shift`, `ctrl`, `alt` / `option`. Keys: `a`–`z`, `0`–`9`, `f1`–`f20`, `space`, `return`, `tab`, `esc`, arrows. Examples: `cmd+shift+s`, `ctrl+alt+f10`.

## Limits

- Protected / DRM content may record as black frames (OS policy).
- Very new macOS SDKs may show deprecation warnings from AVFoundation; capture still works.

