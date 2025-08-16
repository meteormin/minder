package commands

import (
	"fmt"

	"github.com/meteormin/minder"
)

func changeDirectory(c *minder.Context, dest string) (string, error) {
	fp, err := pathToAbs(c, dest)
	if err != nil {
		return "", err
	}
	c.Set("filePath", fp)
	c.Container().RefreshSideBar()
	return fmt.Sprintf("cd: %s", fp), nil
}
