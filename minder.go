package minder

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

type Context struct {
	ctx       context.Context
	window    fyne.Window
	container *LayoutContainer
}

func (c *Context) Set(key string, value any) {
	c.ctx = context.WithValue(c.ctx, key, value)
}

func (c *Context) Get(key string) any {
	return c.ctx.Value(key)
}

func (c *Context) Window() fyne.Window {
	return c.window
}

func (c *Context) Container() *LayoutContainer {
	return c.container
}

type LayoutContainer struct {
	side   *SideContainer
	main   *MainContainer
	bottom *BottomContainer
}

type SideContainer struct {
	c     *fyne.Container
	build func() fyne.CanvasObject
}

type MainContainer struct {
	c     *fyne.Container
	build func() fyne.CanvasObject
}

type BottomContainer struct {
	c     *fyne.Container
	build func() fyne.CanvasObject
}

func (c *LayoutContainer) SideBar(buildFunc func() fyne.CanvasObject) {
	c.side.build = buildFunc
	c.RefreshSideBar()
}

func (c *LayoutContainer) MainFrame(buildFunc func() fyne.CanvasObject) {
	c.main.build = buildFunc
	c.RefreshMainFrame()
}

func (c *LayoutContainer) Bottom(buildFunc func() fyne.CanvasObject) {
	c.bottom.build = buildFunc
	c.RefreshBottom()
}

func (c *LayoutContainer) RefreshSideBar() {
	o := c.side.build()
	if o != nil {
		c.side.c.Objects = []fyne.CanvasObject{container.NewPadded(container.NewScroll(o))}
	}
	c.side.c.Refresh()
}

func (c *LayoutContainer) RefreshMainFrame() {
	o := c.main.build()
	if o != nil {
		c.main.c.Objects = []fyne.CanvasObject{container.NewPadded(container.NewScroll(o))}
	}
	c.main.c.Refresh()
}

func (c *LayoutContainer) RefreshBottom() {
	o := c.bottom.build()
	if o != nil {
		c.bottom.c.Objects = []fyne.CanvasObject{container.NewPadded(container.NewScroll(o))}
	}
	c.bottom.c.Refresh()
}

func mainContainer(c *Context) {
	splitH := container.NewHSplit(c.container.side.c, c.container.main.c)
	splitH.SetOffset(0.2)

	splitV := container.NewVSplit(splitH, c.container.bottom.c)
	splitV.SetOffset(0.8)

	c.window.SetContent(splitV)
}

func New(config Config) (*Context, error) {
	if err := ValidConfig(config); err != nil {
		return nil, err
	}

	a := app.New()
	w := a.NewWindow("Minder")
	w.Resize(config.WindowSize)

	c := &Context{
		ctx:    context.Background(),
		window: w,
		container: &LayoutContainer{
			side:   &SideContainer{c: container.NewStack()},
			main:   &MainContainer{c: container.NewStack()},
			bottom: &BottomContainer{c: container.NewStack()},
		},
	}
	c.Set("filePath", config.BasePath)
	c.Set("logger", config.Logger)

	mainContainer(c)

	return c, nil
}
