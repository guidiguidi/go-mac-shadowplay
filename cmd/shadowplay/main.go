//go:build darwin

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/guidiguidi/go-mac-shadowplay/internal/config"
	"github.com/guidiguidi/go-mac-shadowplay/internal/recorder"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  %s record -o <file.mov>   Record until Ctrl+C
  %s buffer [-config path]  Instant replay buffer (Cmd+Shift+S saves last clip)

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

func cmdRecord(args []string) {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	out := fs.String("o", "", "output movie path (.mov or .mp4)")
	cfgPath := fs.String("config", "", "optional YAML config (for defaults only)")
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

	log.Println("Recording to", *out, "— Ctrl+C to stop")
	if err := rec.StartRecording(*out); err != nil {
		log.Fatal(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	if err := rec.StopRecording(); err != nil {
		log.Fatal(err)
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

	hkSave := hotkey.New([]hotkey.Modifier{hotkey.ModCmd, hotkey.ModShift}, hotkey.KeyS)
	if err := hkSave.Register(); err != nil {
		log.Fatal("register save hotkey:", err)
	}

	go func() {
		for range hkSave.Keydown() {
			if _, err := rec.SaveClip(); err != nil {
				log.Println("save clip:", err)
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	_ = hkSave.Unregister()
	if err := rec.StopBuffer(); err != nil {
		log.Println("stop buffer:", err)
	}
	log.Println("Exiting.")
}
