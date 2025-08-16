package commands

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/meteormin/minder"
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
		Exec: func(_ *minder.Context, _ []string) (string, error) {
			return help()
		},
	}

	cmdExit = Cmd{
		Name:  "exit",
		Usage: "exit",
		Exec: func(c *minder.Context, _ []string) (string, error) {
			return exit(c)
		},
	}
)

type Cmd struct {
	Name  string
	Args  []string
	Usage string
	Exec  func(c *minder.Context, args []string) (string, error)
}

func (cmd Cmd) history(c *minder.Context) {
	logger, _ := c.Get("logger").(*slog.Logger)
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

func help() (string, error) {
	sb := strings.Builder{}
	sb.WriteString("Usage: COMMAND [ARG...]\n\n")
	sb.WriteString("Available Commands:\n")
	for name, c := range commands {
		var usage string
		if c.Usage != "" {
			usage = c.Usage
		} else {
			usage = name + " " + strings.Join(c.Args, " ")
		}
		sb.WriteString("  " + usage + "\n")
	}
	return sb.String(), nil
}

func exit(c *minder.Context) (string, error) {
	c.Window().Close()
	return "", nil
}

func Call(c *minder.Context, cmd string) (string, error) {
	args := parseArgs(cmd)
	args.history(c)
	return args.Exec(c, args.Args)
}
