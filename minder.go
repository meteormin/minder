package minder

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"github.com/meteormin/minder/components"
)

type Store struct {
	Pathfinder  components.PathfinderState
	PreviewPath binding.String
	Terminal    components.TerminalState
}

type Context struct {
	store  *Store
	logger *slog.Logger
	window fyne.Window
	layout *Layout
}

func (c *Context) Store() *Store {
	return c.store
}

func (c *Context) Logger() *slog.Logger {
	return c.logger
}

func (c *Context) Window() fyne.Window {
	return c.window
}

func (c *Context) Layout() *Layout {
	return c.layout
}

type Layout struct {
	sideBar   *LayoutContainer
	mainFrame *LayoutContainer
	bottom    *LayoutContainer
}

type LayoutContainer struct {
	container *fyne.Container
	render    func()
}

func (l *Layout) SideBar() *fyne.Container {
	return l.sideBar.container
}

func (l *Layout) SetSideBar(buildFunc func() fyne.CanvasObject) {
	l.sideBar.render = func() {
		o := buildFunc()
		if o != nil {
			l.sideBar.container.Objects = []fyne.CanvasObject{container.NewScroll(o)}
		}
		l.RefreshSideBar()
	}
	l.sideBar.render()
}

func (l *Layout) MainFrame() *fyne.Container {
	return l.mainFrame.container
}

func (l *Layout) SetMainFrame(buildFunc func() fyne.CanvasObject) {
	l.mainFrame.render = func() {
		o := buildFunc()
		if o != nil {
			l.mainFrame.container.Objects = []fyne.CanvasObject{container.NewScroll(o)}
		}
		l.RefreshMainFrame()
	}
	l.mainFrame.render()
}

func (l *Layout) Bottom() *fyne.Container {
	return l.bottom.container
}

func (l *Layout) SetBottom(buildFunc func() fyne.CanvasObject) {
	l.bottom.render = func() {
		o := buildFunc()
		if o != nil {
			l.bottom.container.Objects = []fyne.CanvasObject{container.NewScroll(o)}
		}
		l.RefreshBottom()
	}
	l.bottom.render()
}

func (l *Layout) RenderSideBar() {
	l.sideBar.render()
}

func (l *Layout) RenderMainFrame() {
	l.mainFrame.render()
}

func (l *Layout) RenderBottom() {
	l.bottom.render()
}

func (l *Layout) RefreshSideBar() {
	fyne.Do(func() {
		l.sideBar.container.Refresh()
	})
}

func (l *Layout) RefreshMainFrame() {
	fyne.Do(func() {
		l.mainFrame.container.Refresh()
	})
}

func (l *Layout) RefreshBottom() {
	fyne.Do(func() {
		l.bottom.container.Refresh()
	})
}

func mainContainer(c *Context) {
	splitH := container.NewHSplit(c.layout.sideBar.container, c.layout.mainFrame.container)
	splitH.SetOffset(0.2)

	splitV := container.NewVSplit(splitH, c.layout.bottom.container)
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

	store := &Store{
		Pathfinder: components.PathfinderState{
			CurrentDir: binding.NewString(),
			ShowHidden: binding.NewBool(),
		},
		PreviewPath: binding.NewString(),
		Terminal: components.TerminalState{
			Input: binding.NewString(),
		},
	}

	err := store.Pathfinder.CurrentDir.Set(config.BasePath)
	if err != nil {
		return nil, err
	}

	c := &Context{
		store:  store,
		window: w,
		logger: config.Logger,
		layout: &Layout{
			sideBar:   &LayoutContainer{container: container.NewStack()},
			mainFrame: &LayoutContainer{container: container.NewStack()},
			bottom:    &LayoutContainer{container: container.NewStack()},
		},
	}

	mainContainer(c)

	return c, nil
}
