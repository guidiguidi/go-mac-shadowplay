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

**Manual recording** (until Ctrl+C):

```bash
./shadowplay record -o ~/Movies/capture.mov
```

**Instant replay buffer** — keeps rolling segment files under `temp_dir`, trims by `buffer_minutes`:

```bash
./shadowplay buffer
./shadowplay buffer -config ./shadowplay.example.yaml
```

While buffer mode is running:

- **⌘⇧S** — export the last `clip_seconds` (from config, default 30s) to `output_dir` as `clip_YYYYMMDD_HHMMSS.mp4`
- **Ctrl+C** — stop and exit

## Configuration

Optional YAML fields (see `shadowplay.example.yaml`):

| Field | Meaning |
|-------|---------|
| `buffer_minutes` | Approximate rolling history kept on disk |
| `clip_seconds` | Length of the file written when you press ⌘⇧S |
| `segment_seconds` | Length of each internal segment file |
| `temp_dir` | Where segment `.mov` files are stored |
| `output_dir` | Where exported clips are saved |

Hotkey strings in YAML are reserved for future use; **⌘⇧S** is fixed in code for now.

## Limits

- Protected / DRM content may record as black frames (OS policy).
- Very new macOS SDKs may show deprecation warnings from AVFoundation; capture still works.

