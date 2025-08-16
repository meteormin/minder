package commands

import (
	"fmt"
	"os"

	"github.com/meteormin/minder"
)

var cmdTouch = Cmd{
	Name: "touch",
	Args: []string{"<dst>"},
	Exec: func(c *minder.Context, args []string) (string, error) {
		return handleTouch(c, args[0])
	},
}

func handleTouch(c *minder.Context, dst string) (string, error) {
	fp, err := pathToAbs(c, dst)
	if err != nil {
		return "", err
	}

	_, err = os.Create(fp)
	if err != nil {
		return "", err
	}

	c.Container().RefreshSideBar()

	return fmt.Sprintf("touch: %s", fp), nil
}
