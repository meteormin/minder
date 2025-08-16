package components

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/theme"
	markdown "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/meteormin/minder"
	_ "golang.org/x/image/webp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Previewer struct {
	logger            *slog.Logger
	maxTextRenderSize int64
}

func (p *Previewer) RenderFile(path string) (fyne.CanvasObject, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var rendered fyne.CanvasObject
	var err error
	switch ext {
	case ".md", ".markdown", ".html", ".htm":
		// md/html은 “보기↔편집 토글” 문서 렌더러로
		rendered, err = p.renderSmartText(path)
	default:
		r, imgErr := p.asImageReader(path)
		if imgErr == nil {
			// 이미지: 버튼/툴바 없이 뷰어 전용
			rendered, err = p.renderImage(r, path)
		} else if p.isTextRenderable(path) {
			// 일반 텍스트: 보기↔편집 토글
			rendered, err = p.renderSmartText(path)
		} else {
			rendered = container.NewCenter(widget.NewLabel(imgErr.Error()))
		}
	}

	card := widget.NewCard(filepath.Base(path), ext, rendered)
	return container.NewPadded(card), err
}

func (p *Previewer) asImageReader(path string) (io.Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(data)
	// 시그니처만 읽어 포맷 판별
	_, format, err := image.DecodeConfig(r)
	if err != nil {
		return nil, err
	}
	if format == "" {
		return nil, errors.New("invalid image format")
	}
	_, err = r.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (p *Previewer) isTextRenderable(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		p.logger.Error("failed stat file", "err", err)
		return false
	}

	if s.Size() > p.maxTextRenderSize {
		return false
	}

	return true
}

func (p *Previewer) renderImage(r io.Reader, path string) (fyne.CanvasObject, error) {
	img := canvas.NewImageFromReader(r, path)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(800, 600))
	return container.NewPadded(container.NewCenter(img)), nil
}

func (p *Previewer) renderSmartText(path string) (fyne.CanvasObject, error) {
	st, err := os.Stat(path)
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error())), err
	}
	if st.IsDir() {
		return container.NewCenter(widget.NewLabel("this is a directory")), nil
	}

	origBytes, err := os.ReadFile(path)
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error())), err
	}
	origText := string(origBytes)
	ext := strings.ToLower(filepath.Ext(path))

	// --- 에디터(편집 모드) ---
	editor := widget.NewMultiLineEntry()
	editor.SetMinRowsVisible(10)
	editor.SetText(origText)
	editor.Disable() // 시작은 보기 모드

	// --- 뷰어(보기 모드) ---
	buildViewer := func(src string) fyne.CanvasObject {
		switch ext {
		case ".md", ".markdown":
			rt := widget.NewRichTextFromMarkdown(src)
			rt.Wrapping = fyne.TextWrapWord
			return container.NewVScroll(rt)
		case ".html", ".htm":
			conv := markdown.NewConverter("", true, nil)
			md, err := conv.ConvertString(src)
			if err != nil || md == "" {
				// 실패하면 코드블록으로 보여줌
				rt := widget.NewRichTextFromMarkdown("```\n" + src + "\n```")
				rt.Wrapping = fyne.TextWrapWord
				return container.NewVScroll(rt)
			}
			rt := widget.NewRichTextFromMarkdown(md)
			rt.Wrapping = fyne.TextWrapWord
			return container.NewVScroll(rt)
		default:
			// 일반 텍스트는 읽기 전용 뷰 (RichText 코드블록)
			rt := widget.NewRichTextFromMarkdown("```text\n" + src + "\n```")
			rt.Wrapping = fyne.TextWrapWord
			return container.NewVScroll(rt)
		}
	}

	view := buildViewer(origText)
	viewScroll := view // 이름만 남겨둠

	// --- 스택: [0]=뷰어, [1]=에디터 ---
	stack := container.NewStack(viewScroll, container.NewVScroll(editor))
	// 시작은 보기 모드이므로 에디터 숨김
	stack.Objects[1].Hide()

	// --- 상단/하단 툴바 ---
	// 보기 모드 툴바: [Edit]
	editBtn := widget.NewButtonWithIcon("edit", theme.DocumentCreateIcon(), func() {
		// 보기→편집으로 전환
		stack.Objects[0].Hide()
		stack.Objects[1].Show()
		editor.Enable()
		editor.FocusGained()
	})
	viewBar := container.NewHBox(editBtn)

	// 편집 모드 툴바: [Save][Cancel]
	saveBtn := widget.NewButtonWithIcon("save", theme.DocumentSaveIcon(), func() {
		newText := editor.Text
		// 저장
		if writeErr := os.WriteFile(path, []byte(newText), 0644); writeErr != nil {
			if p.logger != nil {
				p.logger.Error("failed save file", "err", writeErr)
			}
			return
		}
		// 리렌더 후 보기 모드로 복귀
		newView := buildViewer(newText)
		stack.Objects[0] = newView
		stack.Objects[1].Hide()
		stack.Objects[0].Show()
		editor.Disable()
		stack.Refresh()
	})

	cancelBtn := widget.NewButtonWithIcon("cancel", theme.ContentClearIcon(), func() {
		// 변경 취소하고 원본으로 복귀
		editor.SetText(origText)
		stack.Objects[1].Hide()
		stack.Objects[0].Show()
		editor.Disable()
	})

	// 에디터 변경 시에만 저장 활성화(선택)
	saveBtn.Disable()
	editor.OnChanged = func(s string) {
		if s != origText {
			if saveBtn.Disabled() {
				saveBtn.Enable()
			}
		} else {
			if !saveBtn.Disabled() {
				saveBtn.Disable()
			}
		}
	}
	editBar := container.NewHBox(saveBtn, cancelBtn)
	editBar.Hide() // 시작은 보기 모드

	// 모드에 따라 툴바 전환
	showViewBar := func() {
		editBar.Hide()
		viewBar.Show()
	}
	showEditBar := func() {
		viewBar.Hide()
		editBar.Show()
	}
	editBtn.OnTapped = func() { // 버튼 핸들러 교체(동일 동작)
		stack.Objects[0].Hide()
		stack.Objects[1].Show()
		editor.Enable()
		editor.FocusGained()
		showEditBar()
	}

	// 위에서 정의됨
	cancelBtn.OnTapped = func() {
		editor.SetText(origText)
		stack.Objects[1].Hide()
		stack.Objects[0].Show()
		editor.Disable()
		showViewBar()
	}

	// 전체 레이아웃: 상단 툴바는 모드에 따라 교체, 중앙은 stack
	top := container.NewStack(viewBar, editBar) // 두 바를 겹쳐두고 Hide/Show로 전환
	root := container.NewBorder(top, nil, nil, nil, stack)
	return root, nil
}

func Preview(c *minder.Context) fyne.CanvasObject {
	logger, _ := c.Get("logger").(*slog.Logger)
	selectedFile, _ := c.Get("selectedFile").(string)
	p := &Previewer{
		logger:            logger,
		maxTextRenderSize: 1024 * 1024, // 1MB
	}

	if selectedFile == "" {
		return container.NewPadded(container.NewCenter(widget.NewLabel("no file selected")))
	}

	rendered, err := p.RenderFile(selectedFile)
	if err != nil {
		logger.Error("failed preview", "err", err)
	}

	return container.NewPadded(rendered)
}
