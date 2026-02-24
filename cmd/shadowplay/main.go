package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/guidiguidi/go-mac-shadowplay/internal/hotkey"
	"github.com/guidiguidi/go-mac-shadowplay/internal/recorder"
)

func sendNotification(title, message string) {
	script := fmt.Sprintf("display notification %q with title %q", message, title)
	exec.Command("osascript", "-e", script).Run()
}

func main() {
	r, err := recorder.NewShadowRecorder(3600)
	if err != nil {
		fmt.Printf("❌ Failed to initialize recorder: %v\n", err)
		return
	}

	recordingActive := false

	h := hotkey.NewHotkeyHandler(
		func() { // Toggle
			if recordingActive {
				fmt.Println("\n⏹️  Stopping Shadowplay...")
				r.Stop()
				recordingActive = false
				sendNotification("Shadowplay", "Recording Stopped")
			} else {
				fmt.Println("\n⏺️  Starting Shadowplay (Rolling Buffer)...")
				if err := r.Start(); err != nil {
					fmt.Printf("❌ Error starting recorder: %v\n", err)
					return
				}
				recordingActive = true
				sendNotification("Shadowplay", "Recording Started")
			}
		},
		func() { // Save
			if !recordingActive {
				fmt.Println("\n⚠️  Start recording first (Opt+Shift+R)")
				return
			}
			timestamp := os.Getenv("HOME") + "/Desktop/shadowplay-clip.mp4"
			fmt.Printf("\n💾 Saving clip to %s...\n", timestamp)
			r.Save(timestamp)
			sendNotification("Shadowplay", "Clip Saved to Desktop!")
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\n👋 Exiting...")
		cancel()
	}()

	fmt.Println("🚀 Shadowplay PRO v3.0 (RobotGo Engine)")
	fmt.Println("   Hotkeys: Option+Shift+R / Option+Shift+S")
	
	h.Listen(ctx)
	r.Stop()
}
