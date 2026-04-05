package recorder

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/guidiguidi/go-mac-shadowplay/internal/config"
	"github.com/guidiguidi/go-mac-shadowplay/internal/native"
)

type Recorder struct {
	cfg config.Config
}

func New(cfg config.Config) *Recorder { return &Recorder{cfg: cfg} }

// SetConfig updates paths and timing used by SaveClip / buffer (apply while stopped for hotkeys).
func (r *Recorder) SetConfig(cfg config.Config) { r.cfg = cfg }

func (r *Recorder) StartRecording(outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("output path required")
	}
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return native.StartRecording(outputPath)
}

func (r *Recorder) StopRecording() error {
	return native.StopRecording()
}

func (r *Recorder) StartBuffer() error {
	if err := os.MkdirAll(r.cfg.TempDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(r.cfg.OutputDir, 0o755); err != nil {
		return err
	}

	native.SetSegmentClosedHook(func(path string) {
		log.Println("segment closed:", filepath.Base(path))
	})

	seg := r.cfg.SegmentSeconds
	if seg <= 0 {
		seg = 3
	}
	maxSec := float64(r.cfg.BufferMinutes * 60)
	if maxSec <= 0 {
		maxSec = 600
	}

	log.Printf("buffer temp=%s segment=%.1fs max≈%dm — Cmd+Shift+S save clip, Ctrl+C exit",
		r.cfg.TempDir, seg, r.cfg.BufferMinutes)
	return native.RollingStart(r.cfg.TempDir, seg, maxSec)
}

func (r *Recorder) StopBuffer() error {
	return native.RollingStop()
}

func (r *Recorder) SaveClip() (string, error) {
	if err := os.MkdirAll(r.cfg.OutputDir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("clip_%s.mp4", time.Now().Format("20060102_150405"))
	out := filepath.Join(r.cfg.OutputDir, name)
	sec := r.cfg.ClipSeconds
	if sec <= 0 {
		sec = 30
	}
	if err := native.ExportLast(out, float64(sec)); err != nil {
		return "", err
	}
	log.Println("saved clip:", out)
	return out, nil
}
