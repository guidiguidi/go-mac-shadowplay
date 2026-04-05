//go:build darwin

package native

import "github.com/guidiguidi/go-mac-shadowplay/internal/config"

// GUICallbacks holds the functions the menu bar invokes.
// Set them before calling RunGUI.
type GUICallbacks struct {
	OnStartBuffer func()
	OnStopBuffer  func()
	OnSaveClip    func()
	OnOpenFolder  func()
	OnQuit        func()
	// If set, preferences are blocked while the buffer is running.
	IsBufferActive func() bool
	// Called after preferences are saved to disk (apply to runner).
	OnConfigSaved func(c config.Config)
}
