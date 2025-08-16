package commands

import (
	"fmt"

	"github.com/meteormin/minder"
)

var cmdCd = Cmd{
	Name: "cd",
	Args: []string{"<dst>"},
	Exec: func(c *minder.Context, args []string) (string, error) {
		return handleChangeDirectory(c, args[0])
	},
}

func handleChangeDirectory(c *minder.Context, dst string) (string, error) {
	fp, err := pathToAbs(c, dst)
	if err != nil {
		return "", err
	}
	c.Set("filePath", fp)
	c.Container().RefreshSideBar()
	return fmt.Sprintf("cd: %s", fp), nil
}
