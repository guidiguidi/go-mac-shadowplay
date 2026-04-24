//go:build darwin

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/guidiguidi/go-mac-shadowplay/internal/config"
	"github.com/guidiguidi/go-mac-shadowplay/internal/hotkeyparse"
	"github.com/guidiguidi/go-mac-shadowplay/internal/native"
	"github.com/guidiguidi/go-mac-shadowplay/internal/recorder"
	"github.com/guidiguidi/go-mac-shadowplay/internal/runner"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  %[1]s record -o <file.mov>   Record until Ctrl+C or record_hotkey
  %[1]s buffer [-config path]  Buffer mode (CLI): save_hotkey saves clip
  %[1]s gui    [-config path]  Menu bar app with buffer controls

`, os.Args[0])
	os.Exit(2)
}

func main() {
	mainthread.Init(realMain)
}

func realMain() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "record":
		cmdRecord(os.Args[2:])
	case "buffer":
		cmdBuffer(os.Args[2:])
	case "gui":
		cmdGUI(os.Args[2:])
	default:
		usage()
	}
}

func loadCfg(path string) config.Config {
	c, err := config.Load(path)
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func cmdRecord(args []string) {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	out := fs.String("o", "", "output movie path (.mov or .mp4)")
	cfgPath := fs.String("config", "", "optional YAML config (uses record_hotkey to stop)")
	_ = fs.Parse(args)

	if *out == "" {
		fs.Usage()
		os.Exit(2)
	}
	cfg := config.Default()
	if *cfgPath != "" {
		cfg = loadCfg(*cfgPath)
	}
	rec := recorder.New(cfg)

	mods, key, err := hotkeyparse.Parse(cfg.RecordHotkey)
	if err != nil {
		log.Fatalf("record_hotkey %q: %v", cfg.RecordHotkey, err)
	}
	hkStop := hotkey.New(mods, key)
	if err := hkStop.Register(); err != nil {
		log.Fatal("register stop hotkey:", err)
	}
	defer func() { _ = hkStop.Unregister() }()

	log.Printf("Recording to %s — Ctrl+C or %q to stop", *out, cfg.RecordHotkey)
	if err := rec.StartRecording(*out); err != nil {
		log.Fatal(err)
	}

	var once sync.Once
	stop := func() {
		once.Do(func() {
			if err := rec.StopRecording(); err != nil {
				log.Println("stop:", err)
			}
		})
	}

	done := make(chan struct{})
	go func() {
		<-hkStop.Keydown()
		stop()
		close(done)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
		stop()
	case <-done:
	}
	log.Println("Stopped.")
}

func cmdBuffer(args []string) {
	fs := flag.NewFlagSet("buffer", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to YAML config")
	_ = fs.Parse(args)

	cfg := config.Default()
	if *cfgPath != "" {
		cfg = loadCfg(*cfgPath)
	}

	br := runner.NewBufferRunner(cfg)
	if err := br.Start(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Buffer: %q → save clip, Ctrl+C exit", cfg.SaveHotkey)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	if err := br.Stop(); err != nil {
		log.Println("stop buffer:", err)
	}
	log.Println("Exiting.")
}

func cmdGUI(args []string) {
	fs := flag.NewFlagSet("gui", flag.ExitOnError)
	cfgPath := fs.String("config", "", "path to YAML config")
	_ = fs.Parse(args)

	cfg := config.Default()
	if *cfgPath != "" {
		cfg = loadCfg(*cfgPath)
	}

	br := runner.NewBufferRunner(cfg)

	native.RunGUI(*cfgPath, cfg, native.GUICallbacks{
		IsBufferActive: br.IsActive,
		OnStartBuffer: func() {
			if err := br.Start(); err != nil {
				log.Println("start buffer:", err)
				native.GUINotify("ShadowPlay Error", "Failed to start buffer")
				return
			}
			native.GUISetBuffering(true)
			native.GUINotify("ShadowPlay", "Buffer started")
			log.Println("buffer started")
		},
		OnStopBuffer: func() {
			if err := br.Stop(); err != nil {
				log.Println("stop buffer:", err)
			}
			native.GUISetBuffering(false)
			native.GUINotify("ShadowPlay", "Buffer stopped")
			log.Println("buffer stopped")
		},
		OnSaveClip: func() {
			path, err := br.SaveClip()
			if err != nil {
				log.Println("save clip:", err)
				native.GUINotify("ShadowPlay Error", err.Error())
				return
			}
			native.GUINotify("Clip Saved", filepath.Base(path))
		},
		OnOpenFolder: func() {
			_ = exec.Command("open", br.OutputDir()).Run()
		},
		OnConfigSaved: func(c config.Config) {
			br.SetConfig(c)
		},
		OnQuit: func() {
			if br.IsActive() {
				_ = br.Stop()
			}
			native.GUIQuit()
		},
	})
}
