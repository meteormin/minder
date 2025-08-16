package commands

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/meteormin/minder"
)

var cmdRm = Cmd{
	Name: "rm",
	Args: []string{"<src>", "<dst>"},
	Exec: func(c *minder.Context, args []string) (string, error) {
		return handleRemove(c, args[0])
	},
}

type rmMode int

const (
	rmAsk rmMode = iota
	rmDeleteAll
	rmSkipAll
)

type remover struct {
	window fyne.Window
	logger *slog.Logger
	mode   rmMode // 사용자가 "모두" 선택 시 상태 고정
}

// removeEntry: 패턴/"/." 처리하는 엔트리(래퍼)
func (r *remover) removeEntry(srcSpec string) error {
	// 1) "aDir/." → 내용만 삭제
	if dir, ok := asDotContents(srcSpec); ok {
		return r.removeDirContents(dir)
	}

	// 2) 글롭 확장 (*, ?, [])
	srcs, err := expandPattern(srcSpec)
	if err != nil {
		return err
	}

	// 3) 각각 삭제
	for _, s := range srcs {
		// 글롭 결과에 "/."가 남아있을 가능성은 거의 없지만, 안전상 한 번 더 체크
		if dir, ok := asDotContents(s); ok {
			if err := r.removeDirContents(dir); err != nil {
				return err
			}
			continue
		}
		if err := r.remove(s); err != nil {
			return err
		}
	}
	return nil
}

// remove: 단일 경로 삭제 (파일/디렉터리)
func (r *remover) remove(path string) error {
	if isDangerousRoot(path) {
		return fmt.Errorf("refuse to remove dangerous path: %s", path)
	}

	fi, err := os.Lstat(path)
	if err != nil {
		// 이미 없음 → rm 기본 동작처럼 에러로 돌려줌(원하면 무시 가능)
		return err
	}

	// 사용자 확인(모드에 따라 묻지 않거나/한 번만 모두 적용)
	action, err := r.resolveRemoveConfirm(path, fi.IsDir())
	if err != nil {
		return err
	}
	if action == "skip" {
		return nil
	}

	if fi.IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

// removeDirContents: 디렉터리의 "내용만" 삭제 (디렉터리 자신은 보존)
func (r *remover) removeDirContents(dir string) error {
	if isDangerousRoot(dir) {
		return fmt.Errorf("refuse to clear dangerous dir: %s", dir)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		p := filepath.Join(dir, e.Name())
		if err := r.remove(p); err != nil {
			return err
		}
	}
	return nil
}

func (r *remover) resolveRemoveConfirm(target string, isDir bool) (string, error) {
	switch r.mode {
	case rmDeleteAll:
		return "delete", nil
	case rmSkipAll:
		return "skip", nil
	default:
		ch := make(chan string, 1)
		var dd dialog.Dialog

		fyne.Do(func() {
			kind := "file"
			if isDir {
				kind = "directory"
			}

			title := "Confirm delete"
			msg := widget.NewLabel(fmt.Sprintf("Delete this %s?\n%s", kind, target))

			btnSkip := widget.NewButton("skip", func() { ch <- "skip"; dd.Hide() })
			btnDel := widget.NewButton("delete", func() { ch <- "delete"; dd.Hide() })

			grid := container.NewGridWithColumns(2, btnSkip, btnDel)
			content := container.NewVBox(msg, widget.NewSeparator(), grid)

			dd = dialog.NewCustomWithoutButtons(title, content, r.window)
			dd.Show()
		})

		return <-ch, nil
	}
}

// rm -rf 실수 방지 가드 (원하면 완화 가능)
func isDangerousRoot(p string) bool {
	c := filepath.Clean(p)
	if c == "" || c == "." || c == string(filepath.Separator) {
		return true
	}
	// 홈 루트 삭제 방지 로직
	if u, _ := os.UserHomeDir(); u != "" {
		abs, _ := filepath.Abs(c)
		ua, _ := filepath.Abs(u)
		if abs == ua {
			return true
		}
	}
	return false
}

// 시그니처를 handleCopy와 "같이" 맞추기 위해 dst는 무시합니다.
func handleRemove(c *minder.Context, src string) (string, error) {
	logger := c.Get("logger").(*slog.Logger)
	rm := &remover{
		window: c.Window(),
		logger: logger,
		mode:   rmAsk,
	}

	absSrc, err := pathToAbs(c, src)
	if err != nil {
		logger.Error("failed path to abs", "src", src, "err", err)
		return "", err
	}

	// ⚠️ 반드시 고루틴에서 실행하고, 오류 표시는 fyne.Do(dialog...)로
	if err = rm.removeEntry(absSrc); err != nil {
		return "", err
	}

	c.Container().RefreshSideBar()

	return fmt.Sprintf("rm: %s", absSrc), nil
}
