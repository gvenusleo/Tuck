package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const (
	MinFontSize = 8
	MaxFontSize = 48
)

type Config struct {
	Theme    string  `json:"terminal_theme"`
	Font     string  `json:"terminal_font"`
	FontSize float64 `json:"terminal_font_size"`
	Width    float64 `json:"terminal_width"`
	Height   float64 `json:"terminal_height"`
}

func Default() Config {
	return Config{
		Theme:    "Ghostty Default Style Dark",
		Font:     "JetBrainsMono Nerd Font",
		FontSize: 12,
		Width:    720,
		Height:   420,
	}
}

func Path() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(dir) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "tuck", "config.json"), nil
}

func Load(path string) (Config, error) {
	c := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return Default(), err
	}
	if err := c.Validate(); err != nil {
		return Default(), err
	}
	return c, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Theme) == "" {
		return errors.New("终端主题不能为空")
	}
	if strings.TrimSpace(c.Font) == "" {
		return errors.New("终端字体不能为空")
	}
	if math.IsNaN(c.FontSize) || math.IsInf(c.FontSize, 0) || c.FontSize < MinFontSize || c.FontSize > MaxFontSize {
		return fmt.Errorf("终端字号必须在 %d 到 %d 之间", MinFontSize, MaxFontSize)
	}
	for _, size := range []float64{c.Width, c.Height} {
		if math.IsNaN(size) || math.IsInf(size, 0) || size < 1 || math.Trunc(size) != size {
			return errors.New("终端宽高必须为正整数")
		}
	}
	return nil
}

// Save 用同目录的临时文件替换配置，避免写入中断破坏原文件。
func Save(path string, c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
