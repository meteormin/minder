package commands

import "github.com/meteormin/minder"

func clearHistory(c *minder.Context) (string, error) {
	c.Container().RefreshBottom()
	return "", nil
}
