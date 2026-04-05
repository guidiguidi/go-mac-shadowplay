package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BufferMinutes  int     `yaml:"buffer_minutes"`
	ClipSeconds    int     `yaml:"clip_seconds"`
	SegmentSeconds float64 `yaml:"segment_seconds"`
	OutputDir      string  `yaml:"output_dir"`
	TempDir        string  `yaml:"temp_dir"`
	SaveHotkey     string  `yaml:"save_hotkey"`
	RecordHotkey   string  `yaml:"record_hotkey"`
}

func Default() Config {
	home, _ := os.UserHomeDir()
	td := filepath.Join(os.TempDir(), "shadowplay-buffer")
	out := filepath.Join(home, "Movies", "ShadowPlay")
	return Config{
		BufferMinutes:  10,
		ClipSeconds:    30,
		SegmentSeconds: 3,
		OutputDir:      out,
		TempDir:        td,
		SaveHotkey:     "cmd+shift+s",
		RecordHotkey:   "cmd+shift+r",
	}
}

func Load(path string) (Config, error) {
	c := Default()
	if path == "" {
		return c, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return c, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("parse config: %w", err)
	}
	c.OutputDir = expandPath(c.OutputDir)
	c.TempDir = expandPath(c.TempDir)
	return c, nil
}

// Save writes config as YAML. Paths with ~/ are expanded before writing.
func Save(path string, c Config) error {
	if path == "" {
		return fmt.Errorf("config path empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	c.OutputDir = expandPath(c.OutputDir)
	c.TempDir = expandPath(c.TempDir)
	b, err := yaml.Marshal(&c)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func expandPath(s string) string {
	if strings.HasPrefix(s, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return s
		}
		return filepath.Join(home, strings.TrimPrefix(s, "~/"))
	}
	return s
}
