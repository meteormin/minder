package commands

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/meteormin/minder"
)

type Args struct {
	cmd  string
	args []string
}

func (args Args) history(c *minder.Context) {
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
	_, err = fmt.Fprintf(&builder, "[%d] %s ", pid, args.cmd)
	if err != nil {
		logger.Error("failed write command", "cmd", args.cmd, "err", err)
		return
	}

	builder.WriteString(strings.Join(args.args, " "))
	builder.WriteString("\n")
	data := builder.String()
	// file.Write() 메서드를 사용하여 데이터 쓰기
	if _, err = file.WriteString(data); err != nil {
		logger.Error("failed write file", "data", data, "err", err)
	}
}

func parseArgs(cmd string) Args {
	s := strings.Split(cmd, " ")
	return Args{cmd: s[0], args: s[1:]}
}

func handleHelp() (string, error) {
	sb := strings.Builder{}
	sb.WriteString("Usage: COMMAND [ARG...]\n\n")
	sb.WriteString("Available Commands:\n")
	sb.WriteString("  cd <dest>\n")
	sb.WriteString("  clear\n")
	sb.WriteString("  mkdir <dest>\n")
	sb.WriteString("  touch <dest>\n")
	return sb.String(), nil
}

func handleExit(c *minder.Context) (string, error) {
	c.Window().Close()
	return "", nil
}

func Call(c *minder.Context, cmd string) (string, error) {
	args := parseArgs(cmd)
	args.history(c)
	switch args.cmd {
	case "cd":
		return handleCd(c, args.args[0])
	case "clear":
		return handleClear(c)
	case "cp":
		return handleCopy(c, args.args[0], args.args[1])
	case "mkdir":
		return cmdMakeDirectory(c, args.args[0])
	case "mv":
		return handleMove(c, args.args[0], args.args[1])
	case "rm":
		return handleRemove(c, args.args[0])
	case "touch":
		return handleTouch(c, args.args[0])
	case "exit":
		return handleExit(c)
	default:
		return handleHelp()
	}
}
