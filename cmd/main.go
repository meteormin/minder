package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"github.com/meteormin/minder"
	"github.com/meteormin/minder/components"
)

var (
	logPath = "/var/log/minder.log"
	logger  *slog.Logger
)

func init() {
	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logPath = "./minder.log"
		f, err = os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			panic(err)
		}
	}

	w := io.MultiWriter(os.Stdout, f)
	logger = slog.New(slog.NewTextHandler(w, nil))
}

func main() {
	var basePath string
	if len(os.Args) > 1 {
		logger.Info("argument", "arg[1]", os.Args[1])
		basePath = os.Args[1]
		if basePath == "" {
			wd, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			basePath = wd
		}
	}

	absPath, err := filepath.Abs(filepath.Dir(basePath))
	if err != nil {
		panic(err)
	}

	c, err := minder.New(minder.Config{
		Logger:     logger,
		BasePath:   absPath,
		WindowSize: fyne.NewSize(1920, 1200),
	})
	if err != nil {
		logger.Error("failed new minder application", "err", err)
		panic(err)
	}

	c.Layout().SetSideBar(func() fyne.CanvasObject {
		pf := components.NewPathfinder(components.PathfinderConfig{
			FileTreeConfig: components.FileTreeConfig{
				Window:     c.Window(),
				RootDir:    c.Store().Pathfinder.CurrentDir,
				ShowHidden: c.Store().Pathfinder.ShowHidden,
				OnSelected: func(uid string) {
					setErr := c.Store().PreviewPath.Set(uid)
					if setErr != nil {
						c.Logger().Error("failed select file", "err", setErr)
						return
					}
				},
			},
			Logger: c.Logger(),
		})
		return pf.Container
	})

	c.Layout().SetMainFrame(func() fyne.CanvasObject {
		preview := components.NewPreview(components.PreviewConfig{
			Logger: c.Logger(),
			Path:   c.Store().PreviewPath,
		})
		return preview.PreviewPane.Root()
	})

	c.Layout().SetBottom(func() fyne.CanvasObject {
		term := components.NewTerminal(components.TerminalConfig{
			Logger:         c.Logger(),
			Window:         c.Window(),
			Pwd:            c.Store().Pathfinder.CurrentDir,
			Input:          c.Store().Terminal.Input,
			RefreshSideBar: c.Layout().RenderSideBar,
		})
		return term.Container
	})

	c.Window().ShowAndRun()
}
