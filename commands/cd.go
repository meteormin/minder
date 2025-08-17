package commands

import (
	"fmt"
)

var cmdCd = Cmd{
	Name: "cd",
	Args: []string{"<dst>"},
	Exec: func(c *Context, args []string) error {
		return handleChangeDirectory(c, args[0])
	},
}

func handleChangeDirectory(c *Context, dst string) error {
	fp, err := pathToAbs(c, dst)
	if err != nil {
		return err
	}

	err = c.Pwd.Set(fp)
	if err != nil {
		return err
	}

	c.RefreshSideBar()

	_, err = fmt.Fprintf(c.ConsoleBuf, "cd: %s", fp)
	return err
}
