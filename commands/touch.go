package commands

import (
	"fmt"
	"os"

	"github.com/meteormin/minder"
)

func handleTouch(c *minder.Context, dest string) (string, error) {
	fp, err := pathToAbs(c, dest)
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
