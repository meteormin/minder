package minder

import (
	"errors"
	"log/slog"

	"fyne.io/fyne/v2"
)

type Config struct {
	Logger     *slog.Logger
	BasePath   string
	WindowSize fyne.Size
}

func ValidConfig(cfg Config) error {
	if cfg.Logger == nil {
		return errors.New("logger is required")
	}
	if cfg.BasePath == "" {
		return errors.New("base path is required")
	}
	if cfg.WindowSize.Width == 0.0 || cfg.WindowSize.Height == 0.0 {
		return errors.New("window size is required")
	}
	return nil
}
