package commands

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
)

var commands = map[string]Cmd{
	cmdCd.Name:    cmdCd,
	cmdClear.Name: cmdClear,
	cmdTouch.Name: cmdTouch,
	cmdMkdir.Name: cmdMkdir,
	cmdCopy.Name:  cmdCopy,
	cmdMove.Name:  cmdMove,
	cmdRm.Name:    cmdRm,
	cmdExit.Name:  cmdExit,
}

var (
	cmdHelp = Cmd{
		Name:  "help",
		Usage: "help",
		Exec: func(c *Context, _ []string) error {
			return help(c)
		},
	}

	cmdExit = Cmd{
		Name:  "exit",
		Usage: "exit",
		Exec: func(c *Context, _ []string) error {
			return exit(c)
		},
	}
)

type Context struct {
	Logger         *slog.Logger
	Window         fyne.Window
	Pwd            binding.String
	ConsoleBuf     *strings.Builder
	RefreshSideBar func()
}

type Cmd struct {
	Name  string
	Args  []string
	Usage string
	Exec  func(c *Context, args []string) error
}

func (cmd Cmd) history(c *Context) {
	logger := c.Logger
	home, err := os.UserHomeDir()
	if err != nil {
		home, err = os.Getwd()
		if err != nil {
			home = "./"
		}
	}

	pid := os.Getpid()
	fp := filepath.Join(home, ".minder_history")
	file, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("failed open file", "file", fp, "err", err)
		return
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			logger.Error("failed file close", "err", err)
		}
	}(file)

	var builder strings.Builder
	_, err = fmt.Fprintf(&builder, "[%d] %s ", pid, cmd.Name)
	if err != nil {
		logger.Error("failed write command", "cmd", cmd.Name, "err", err)
		return
	}

	builder.WriteString(strings.Join(cmd.Args, " "))
	builder.WriteString("\n")
	data := builder.String()
	// file.Write() 메서드를 사용하여 데이터 쓰기
	if _, err = file.WriteString(data); err != nil {
		logger.Error("failed write file", "data", data, "err", err)
	}
}

func parseArgs(cmd string) Cmd {
	s := strings.Split(cmd, " ")
	c, ok := commands[s[0]]
	if !ok {
		return cmdHelp
	}
	c.Args = s[1:]
	return c
}

func help(c *Context) error {
	if _, err := c.ConsoleBuf.WriteString("Usage: COMMAND [ARG...]\n\n"); err != nil {
		return err
	}
	if _, err := c.ConsoleBuf.WriteString("Available Commands:\n"); err != nil {
		return err
	}

	for name, cmd := range commands {
		var usage string
		if cmd.Usage != "" {
			usage = cmd.Usage
		} else {
			usage = name + " " + strings.Join(cmd.Args, " ")
		}
		_, err := c.ConsoleBuf.WriteString("  " + usage + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func exit(c *Context) error {
	c.Window.Close()
	return nil
}

func Call(c *Context, cmd string) error {
	args := parseArgs(cmd)
	args.history(c)
	return args.Exec(c, args.Args)
}
