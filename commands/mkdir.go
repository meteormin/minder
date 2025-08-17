package commands

import (
	"fmt"
	"os"
)

var cmdMkdir = Cmd{
	Name: "mkdir",
	Args: []string{"<dst>"},
	Exec: func(c *Context, args []string) error {
		return handleMakeDirectory(c, args[0])
	},
}

func handleMakeDirectory(c *Context, dst string) error {
	fp, err := pathToAbs(c, dst)
	if err != nil {
		return err
	}
	err = os.MkdirAll(fp, 0755)
	if err != nil {
		return err
	}

	c.RefreshSideBar()

	_, err = fmt.Fprintf(c.ConsoleBuf, "mkdir %s", fp)
	return err
}
