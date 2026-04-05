//go:build darwin

package runner

import (
	"fmt"
	"log"

	"github.com/guidiguidi/go-mac-shadowplay/internal/config"
	"github.com/guidiguidi/go-mac-shadowplay/internal/hotkeyparse"
	"github.com/guidiguidi/go-mac-shadowplay/internal/recorder"
	"golang.design/x/hotkey"
)

// BufferRunner manages buffer capture and associated global hotkeys.
// Usable from both CLI and GUI modes.
type BufferRunner struct {
	rec    *recorder.Recorder
	cfg    config.Config
	active bool

	hkSave *hotkey.Hotkey
	hkRec  *hotkey.Hotkey
}

func NewBufferRunner(cfg config.Config) *BufferRunner {
	return &BufferRunner{
		rec: recorder.New(cfg),
		cfg: cfg,
	}
}

func (br *BufferRunner) Start() error {
	if br.active {
		return fmt.Errorf("buffer already running")
	}
	if err := br.rec.StartBuffer(); err != nil {
		return err
	}
	br.active = true

	if err := br.registerHotkeys(); err != nil {
		_ = br.rec.StopBuffer()
		br.active = false
		return err
	}
	return nil
}

func (br *BufferRunner) Stop() error {
	if !br.active {
		return nil
	}
	br.active = false
	br.unregisterHotkeys()
	return br.rec.StopBuffer()
}

func (br *BufferRunner) SaveClip() (string, error) {
	if !br.active {
		return "", fmt.Errorf("buffer not running")
	}
	return br.rec.SaveClip()
}

func (br *BufferRunner) IsActive() bool { return br.active }

func (br *BufferRunner) OutputDir() string { return br.cfg.OutputDir }

func (br *BufferRunner) registerHotkeys() error {
	mSave, kSave, err := hotkeyparse.Parse(br.cfg.SaveHotkey)
	if err != nil {
		return fmt.Errorf("save_hotkey %q: %w", br.cfg.SaveHotkey, err)
	}
	br.hkSave = hotkey.New(mSave, kSave)
	if err := br.hkSave.Register(); err != nil {
		return fmt.Errorf("register save hotkey: %w", err)
	}
	go func() {
		for range br.hkSave.Keydown() {
			if _, err := br.rec.SaveClip(); err != nil {
				log.Println("save clip:", err)
			}
		}
	}()

	if !hotkeyparse.SameBinding(br.cfg.SaveHotkey, br.cfg.RecordHotkey) {
		mRec, kRec, err := hotkeyparse.Parse(br.cfg.RecordHotkey)
		if err != nil {
			return fmt.Errorf("record_hotkey %q: %w", br.cfg.RecordHotkey, err)
		}
		br.hkRec = hotkey.New(mRec, kRec)
		if err := br.hkRec.Register(); err != nil {
			return fmt.Errorf("register record hotkey: %w", err)
		}
		go func() {
			for range br.hkRec.Keydown() {
				if _, err := br.rec.SaveClip(); err != nil {
					log.Println("save clip (record hotkey):", err)
				}
			}
		}()
	}
	return nil
}

func (br *BufferRunner) unregisterHotkeys() {
	if br.hkSave != nil {
		_ = br.hkSave.Unregister()
		br.hkSave = nil
	}
	if br.hkRec != nil {
		_ = br.hkRec.Unregister()
		br.hkRec = nil
	}
}
