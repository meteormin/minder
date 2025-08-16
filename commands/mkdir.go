package commands

import (
	"fmt"
	"os"

	"github.com/meteormin/minder"
)

func cmdMakeDirectory(c *minder.Context, dest string) (string, error) {
	fp, err := pathToAbs(c, dest)
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
