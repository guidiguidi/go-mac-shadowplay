//go:build darwin

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/guidiguidi/go-mac-shadowplay/internal/config"
	"github.com/guidiguidi/go-mac-shadowplay/internal/hotkeyparse"
	"github.com/guidiguidi/go-mac-shadowplay/internal/recorder"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  %s record -o <file.mov>   Record until Ctrl+C or record_hotkey from config
  %s buffer [-config path]  Buffer mode: save_hotkey saves clip, record_hotkey can also save (second binding)

`, os.Args[0], os.Args[0])
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

func mustParseHotkey(name, spec string) ([]hotkey.Modifier, hotkey.Key) {
	mods, key, err := hotkeyparse.Parse(spec)
	if err != nil {
		log.Fatalf("%s hotkey %q: %v", name, spec, err)
	}
	return mods, key
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

	mods, key := mustParseHotkey("record_hotkey (stop recording)", cfg.RecordHotkey)
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
	rec := recorder.New(cfg)
	if err := rec.StartBuffer(); err != nil {
		log.Fatal(err)
	}

	mSave, kSave := mustParseHotkey("save_hotkey", cfg.SaveHotkey)
	hkSave := hotkey.New(mSave, kSave)
	if err := hkSave.Register(); err != nil {
		log.Fatal("register save hotkey:", err)
	}
	defer func() { _ = hkSave.Unregister() }()

	go func() {
		for range hkSave.Keydown() {
			if _, err := rec.SaveClip(); err != nil {
				log.Println("save clip:", err)
			}
		}
	}()

	var hkRec *hotkey.Hotkey
	if !hotkeyparse.SameBinding(cfg.SaveHotkey, cfg.RecordHotkey) {
		mRec, kRec := mustParseHotkey("record_hotkey", cfg.RecordHotkey)
		hkRec = hotkey.New(mRec, kRec)
		if err := hkRec.Register(); err != nil {
			log.Fatal("register record hotkey:", err)
		}
		defer func() { _ = hkRec.Unregister() }()
		go func() {
			for range hkRec.Keydown() {
				if _, err := rec.SaveClip(); err != nil {
					log.Println("save clip (record hotkey):", err)
				}
			}
		}()
	}

	log.Printf("Buffer: %q → save clip, Ctrl+C exit", cfg.SaveHotkey)
	if hkRec != nil {
		log.Printf("Also: %q → save clip (same action, second binding)", cfg.RecordHotkey)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	if err := rec.StopBuffer(); err != nil {
		log.Println("stop buffer:", err)
	}
	log.Println("Exiting.")
}
