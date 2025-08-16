package commands

import "github.com/meteormin/minder"

var cmdClear = Cmd{
	Name:  "clear",
	Usage: "clear",
	Exec: func(c *minder.Context, args []string) (string, error) {
		return handleClear(c)
	},
}

func handleClear(c *minder.Context) (string, error) {
	c.Container().RefreshBottom()
	return "", nil
}
