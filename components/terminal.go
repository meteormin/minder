package components

import (
	"log/slog"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/meteormin/minder/commands"
)

type Console struct {
	grid   *widget.TextGrid
	scroll *container.Scroll
	buf    strings.Builder
	mu     sync.Mutex
	cmd    string
}

func (cs *Console) println(line string) {
	cs.mu.Lock()
	cs.buf.WriteString(line)
	cs.buf.WriteByte('\n')
	cs.mu.Unlock()
	text := cs.buf.String()
	fyne.Do(func() {
		cs.grid.SetText(text)
		// 레이아웃 반영 직후 바닥으로
		cs.scroll.ScrollToBottom()
	})
}

func (cs *Console) render() {
	cs.mu.Lock()
	cs.buf.WriteByte('\n')
	cs.mu.Unlock()
	text := cs.buf.String()
	fyne.Do(func() {
		cs.grid.SetText(text)
		// 레이아웃 반영 직후 바닥으로
		cs.scroll.ScrollToBottom()
	})
}

func (cs *Console) command() string {
	return cs.cmd
}

func (cs *Console) handleSubmitted(c *commands.Context) {
	cmd := cs.command()
	cmdErr := commands.Call(c, cmd)
	if cmdErr != nil {
		cs.println(cmdErr.Error())
		return
	}
	cs.render()
}

type TerminalState struct {
	Input binding.String
}

type TerminalConfig struct {
	Pwd            binding.String
	Input          binding.String
	Logger         *slog.Logger
	Window         fyne.Window
	RefreshSideBar func()
}

type Terminal struct {
	State     TerminalState
	Container fyne.CanvasObject
}

func NewTerminal(config TerminalConfig) *Terminal {
	// 히스토리: TextGrid + 바깥 VScroll (Entry 아님)
	grid := widget.NewTextGrid()
	scroll := container.NewVScroll(grid)

	console := &Console{grid: grid, scroll: scroll}
	console.buf.Reset()
	ctx := &commands.Context{
		Pwd:            config.Pwd,
		ConsoleBuf:     &console.buf,
		Logger:         config.Logger,
		Window:         config.Window,
		RefreshSideBar: config.RefreshSideBar,
	}

	// 프롬프트 + 입력
	promptLabel := widget.NewLabel(">")
	prompt := widget.NewEntryWithData(config.Input)
	prompt.SetPlaceHolder("type here and press Enter")
	prompt.OnSubmitted = func(s string) {
		if s == "" {
			return
		}
		// 프롬프트와 함께 즉시 출력 (UI 스레드)
		console.cmd = s
		console.println("> " + s)
		prompt.SetText("")

		// 실제 처리는 고루틴에서, UI 갱신은 console.Println 사용
		go console.handleSubmitted(ctx)
	}

	bottom := container.NewBorder(nil, nil, promptLabel, nil, prompt)
	c := container.NewBorder(nil, bottom, nil, nil, scroll)

	return &Terminal{
		State: TerminalState{
			Input: config.Input,
		},
		Container: c,
	}
}
