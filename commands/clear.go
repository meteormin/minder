package commands

var cmdClear = Cmd{
	Name:  "clear",
	Usage: "clear",
	Exec: func(c *Context, args []string) error {
		return handleClear(c)
	},
}

func handleClear(c *Context) error {
	c.ConsoleBuf.Reset()
	return nil
}
