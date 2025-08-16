package commands

import (
	"fmt"
	"os"

	"github.com/meteormin/minder"
)

var cmdMkdir = Cmd{
	Name: "mkdir",
	Args: []string{"<dst>"},
	Exec: func(c *minder.Context, args []string) (string, error) {
		return handleMakeDirectory(c, args[0])
	},
}

func handleMakeDirectory(c *minder.Context, dst string) (string, error) {
	fp, err := pathToAbs(c, dst)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(fp, 0755)
	if err != nil {
		return "", err
	}

	c.Container().RefreshSideBar()

	return fmt.Sprintf("mkdir %s", fp), nil
}
