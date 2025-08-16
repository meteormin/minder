package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/meteormin/minder"
)

type Console struct {
	entry *widget.Entry
	cmd   string
}

func (c *Console) Command() string {
	return c.cmd
}

func (c *Console) AddMessage(msg string) {
	c.entry.Text += msg + "\n"
	c.entry.Refresh()
}

func Terminal(c *minder.Context, onSubmitted func(c *minder.Context, console *Console)) fyne.CanvasObject {
	history := widget.NewMultiLineEntry()
	history.Wrapping = fyne.TextWrapOff
	history.SetPlaceHolder("") // 플레이스홀더 제거
	history.SetMinRowsVisible(5)
	history.Disable()
	scroll := container.NewVScroll(history)

	prompt := widget.NewLabel(">") // 프롬프트
	input := widget.NewEntry()
	input.SetPlaceHolder("type here and press Enter")
	input.OnSubmitted = func(s string) {
		if s == "" {
			return
		}
		// 기록 추가
		history.Enable()
		history.SetText(history.Text + "> " + s + "\n")
		history.Disable()
		input.SetText("")

		// 최신 줄로 스크롤
		scroll.ScrollToBottom()

		console := &Console{
			entry: history,
			cmd:   s,
		}

		onSubmitted(c, console)
	}

	// 레이아웃
	promptUI := container.NewBorder(nil, nil, prompt, nil, input)
	return container.NewBorder(nil, promptUI, nil, nil, scroll)
}
