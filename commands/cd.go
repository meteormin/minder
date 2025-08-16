package commands

import (
	"fmt"

	"github.com/meteormin/minder"
)

func handleChangeDirectory(c *minder.Context, dst string) (string, error) {
	fp, err := pathToAbs(c, dst)
	if err != nil {
		return "", err
	}
	c.Set("filePath", fp)
	c.Container().RefreshSideBar()
	return fmt.Sprintf("cd: %s", fp), nil
}
