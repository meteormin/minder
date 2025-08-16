package commands

import (
	"os"
	"path/filepath"

	"github.com/meteormin/minder"
)

func pathToAbs(c *minder.Context, dest string) (string, error) {
	var fp string
	if filepath.IsAbs(dest) {
		if _, err := os.Stat(dest); err != nil {
			return "", err
		}
		fp = filepath.Dir(dest)
	}

	currentDir, _ := c.Get("filePath").(string)
	s, err := os.Lstat(currentDir)
	if err != nil {
		return "", err
	}

	if s.IsDir() {
		fp = filepath.Join(currentDir, dest)
	} else {
		fp = filepath.Join(filepath.Dir(currentDir), dest)
	}

	return fp, nil
}
