package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"github.com/meteormin/minder"
	"github.com/meteormin/minder/commands"
	"github.com/meteormin/minder/components"
)

var (
	homeDir string
	logPath = "/var/log/minder.log"
	logger  *slog.Logger
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	homeDir = userHomeDir

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

	c.Container().SideBar(func() fyne.CanvasObject {
		return components.Pathfinder(c, func(c *minder.Context, s string) {
			logger.Info("select file", "selectedFile", s)
			c.Set("selectedFile", s)
			c.Container().RefreshMainFrame()
		})
	})

	c.Container().Bottom(func() fyne.CanvasObject {
		return components.Terminal(c, func(c *minder.Context, console *components.Console) {
			cmd := console.Command()
			rs, cmdErr := commands.Call(c, cmd)
			if cmdErr != nil {
				console.AddMessage(cmdErr.Error())
				return
			}
			console.AddMessage(rs)
		})
	})

	c.Container().MainFrame(func() fyne.CanvasObject {
		return components.Preview(c)
	})

	c.Window().ShowAndRun()
}
