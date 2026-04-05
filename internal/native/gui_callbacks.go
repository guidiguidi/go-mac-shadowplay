//go:build darwin

package native

// GUICallbacks holds the functions the menu bar invokes.
// Set them before calling RunGUI.
type GUICallbacks struct {
	OnStartBuffer func()
	OnStopBuffer  func()
	OnSaveClip    func()
	OnOpenFolder  func()
	OnQuit        func()
}
