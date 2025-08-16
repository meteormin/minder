package commands

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/meteormin/minder"
)

type copier struct {
	window fyne.Window
	logger *slog.Logger
	mode   conflictMode // 사용자가 "모두"를 선택하면 상태 고정
}

// copyEntry 패턴/".", 숨김 포함 여부까지 처리하는 엔트리 포인트
func (c *copier) copyEntry(srcPattern, dst string) error {
	// 1) "aDir/." → 내용만 복사
	if dir, ok := asDotContents(srcPattern); ok {
		return c.copyDirContents(dir, dst)
	}

	// 2) 글롭 확장
	srcs, err := expandPattern(srcPattern)
	if err != nil {
		return err
	}
	// 3) 다중 소스면 목적지는 반드시 디렉터리여야
	if len(srcs) > 1 {
		if di, err := os.Stat(dst); err != nil || !di.IsDir() {
			return fmt.Errorf("target %q is not a directory for multiple sources", dst)
		}
	}
	// 4) 각각 처리
	for _, s := range srcs {
		if dir, ok := asDotContents(s); ok {
			// "aDir/." → 내용만 복사
			if err := c.copyDirContents(dir, dst); err != nil {
				return err
			}
			continue
		}
		if err := c.copy(s, dst); err != nil {
			return err
		}
	}
	return nil
}

// moveEntry mv도 동일 정책
func (c *copier) moveEntry(srcPattern, dst string) error {
	// 1) "aDir/." → 내용만 복사
	if dir, ok := asDotContents(srcPattern); ok {
		return c.copyDirContents(dir, dst)
	}

	// 2) 글롭 확장
	srcs, err := expandPattern(srcPattern)
	if err != nil {
		return err
	}
	if len(srcs) > 1 {
		if di, err := os.Stat(dst); err != nil || !di.IsDir() {
			return fmt.Errorf("target %q is not a directory for multiple sources", dst)
		}
	}
	for _, s := range srcs {
		if dir, ok := asDotContents(s); ok {
			// "aDir/." → 내용만 이동
			if err := c.moveDirContents(dir, dst); err != nil {
				return err
			}
			continue
		}
		if err := c.move(s, dst); err != nil {
			return err
		}
	}
	return nil
}

// moveDirContents: copyDirContents와 대칭
func (c *copier) moveDirContents(srcDir, dstDir string) error {
	ents, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	for _, e := range ents {
		s := filepath.Join(srcDir, e.Name())
		d := filepath.Join(dstDir, e.Name())
		if err := c.move(s, d); err != nil {
			return err
		}
	}
	return nil
}

func (c *copier) copy(src, dst string) error {
	// dst가 디렉터리면 src 베이스 이름으로 붙임
	if di, err := os.Stat(dst); err == nil && di.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return c.copyDir(src, dst)
	}
	return c.copyFile(src, dst)
}

func (c *copier) move(src, dst string) error {
	// dst가 디렉터리면 src 베이스 이름으로 붙임
	if di, err := os.Stat(dst); err == nil && di.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	// 우선 rename
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !isCrossDevice(err) && !shouldFallbackRename(err) {
		// 다른 이유면 그대로 리턴
		return err
	}
	// 폴백: copy(+재귀) → remove
	if err := c.copy(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func (c *copier) copyDir(src, dst string) error {
	// 자기 하위로 복사 금지
	srcAbs, _ := filepath.Abs(src)
	dstAbs, _ := filepath.Abs(dst)
	if isSubpath(dstAbs, srcAbs) {
		return fmt.Errorf("destination is inside source: %s -> %s", src, dst)
	}

	// 최상위 대상이 존재 & 파일이면 충돌 처리
	if st, err := os.Lstat(dst); err == nil && !st.IsDir() {
		action, err := c.resolveConflict(dst)
		if err != nil {
			return err
		}
		if action == "skip" {
			return nil
		}
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		// 파일: 충돌 확인
		if exists(target) {
			act, err := c.resolveConflict(target)
			if err != nil {
				return err
			}
			if act == "skip" {
				return nil
			}
			if err := os.RemoveAll(target); err != nil {
				return err
			}
		}
		return c.copyOneFile(path, target, info.Mode().Perm())
	})
}

func (c *copier) copyFile(src, dst string) error {
	// 대상이 디렉터리라면 파일명 붙임
	if di, err := os.Stat(dst); err == nil && di.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	// 충돌 처리
	if exists(dst) {
		act, err := c.resolveConflict(dst)
		if err != nil {
			return err
		}
		if act == "skip" {
			return nil
		}
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
	}
	// 부모 생성 후 복사
	srcFi, err := os.Stat(src)
	if err != nil {
		return err
	}
	return c.copyOneFile(src, dst, srcFi.Mode().Perm())
}

func (c *copier) copyOneFile(src, dst string, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(in *os.File) {
		inErr := in.Close()
		if inErr != nil {
			c.logger.Error("failed close file", "src", src)
		}
	}(in)

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		outErr := out.Close()
		if outErr != nil {
			c.logger.Error("failed close file", "dst", dst)
		}
	}(out)

	_, err = io.Copy(out, in)
	return err
}

func (c *copier) resolveConflict(dst string) (string, error) {
	switch c.mode {
	case modeOverwriteAll:
		return "overwrite", nil
	case modeSkipAll:
		return "skip", nil
	default:
		// 4-버튼 모달로 물어보기 (UI 스레드에서 생성)
		ch := make(chan string, 1)
		var dd dialog.Dialog

		show := func() {
			title := "already exists file"
			msg := widget.NewLabel(fmt.Sprintf("already exists file:\n%s\noverwrite?", dst))
			btnSkip := widget.NewButton("skip", func() { ch <- "skip"; dd.Hide() })
			btnOverwrite := widget.NewButton("overwrite", func() { ch <- "overwrite"; dd.Hide() })

			grid := container.NewGridWithColumns(2, btnSkip, btnOverwrite)
			content := container.NewVBox(msg, widget.NewSeparator(), grid)

			dd = dialog.NewCustomWithoutButtons(title, content, c.window)
			dd.Show()
		}
		// UI 스레드에서 다이얼로그 띄우기
		fyne.Do(show)

		// 백그라운드(현재 고루틴)에서 사용자 선택 대기
		return <-ch, nil
	}
}

func (c *copier) copyDirContents(srcDir, dstDir string) error {
	ents, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	for _, e := range ents {
		s := filepath.Join(srcDir, e.Name())
		d := filepath.Join(dstDir, e.Name())
		if e.IsDir() {
			if err := c.copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := c.copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func handleCopy(c *minder.Context, src string, dst string) (string, error) {
	logger := c.Get("logger").(*slog.Logger)
	cp := &copier{
		window: c.Window(),
		logger: logger,
		mode:   0,
	}

	absSrc, err := pathToAbs(c, src)
	if err != nil {
		logger.Error("failed path to abs", "src", src)
		return "", err
	}
	absDst, err := pathToAbs(c, dst)
	if err != nil {
		logger.Error("failed path to abs", "dst", dst)
		return "", err
	}

	err = cp.copyEntry(absSrc, absDst)
	if err != nil {
		return "", err
	}

	c.Container().RefreshSideBar()

	return fmt.Sprintf("cp: %s to %s", absSrc, absDst), nil
}

func handleMove(c *minder.Context, src string, dst string) (string, error) {
	logger := c.Get("logger").(*slog.Logger)
	cp := &copier{
		window: c.Window(),
		logger: logger,
		mode:   0,
	}

	absSrc, err := pathToAbs(c, src)
	if err != nil {
		logger.Error("failed path to abs", "src", src)
		return "", err
	}
	absDst, err := pathToAbs(c, dst)
	if err != nil {
		logger.Error("failed path to abs", "dst", dst)
		return "", err
	}

	err = cp.moveEntry(absSrc, absDst)
	if err != nil {
		return "", err
	}

	c.Container().RefreshSideBar()

	return fmt.Sprintf("mv: %s to %s", absSrc, absDst), nil
}
