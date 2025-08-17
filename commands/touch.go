package commands

import (
	"fmt"
	"os"
)

var cmdTouch = Cmd{
	Name: "touch",
	Args: []string{"<dst>"},
	Exec: func(c *Context, args []string) error {
		return handleTouch(c, args[0])
	},
}

func handleTouch(c *Context, dst string) error {
	fp, err := pathToAbs(c, dst)
	if err != nil {
		return err
	}

	_, err = os.Create(fp)
	if err != nil {
		return err
	}

	c.RefreshSideBar()

	_, err = fmt.Fprintf(c.ConsoleBuf, "touch: %s", fp)
	return err
}
