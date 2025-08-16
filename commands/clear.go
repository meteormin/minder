package commands

import "github.com/meteormin/minder"

func handleClear(c *minder.Context) (string, error) {
	c.Container().RefreshBottom()
	return "", nil
}
