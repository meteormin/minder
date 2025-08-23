package commands

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var cmdCopy = Cmd{
	Name: "cp",
	Args: []string{"<src>", "<dst>"},
	Exec: func(c *Context, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("cp: missing argument")
		}
		return handleCopy(c, args[0], args[1])
	},
}

// copyEntry 패턴/".", 숨김 포함 여부까지 처리하는 엔트리 포인트
func copyEntry(c *Context, srcPattern, dst string) error {
	// 1) "aDir/." → 내용만 복사
	if dir, ok := asDotContents(srcPattern); ok {
		return copyDirContents(c, dir, dst)
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
			if err = copyDirContents(c, dir, dst); err != nil {
				return err
			}
			continue
		}
		if err = copyAny(c, s, dst); err != nil {
			return err
		}
	}
	return nil
}

func copyAny(c *Context, src, dst string) error {
	// dst가 디렉터리면 src 베이스 이름으로 붙임
	if di, err := os.Stat(dst); err == nil && di.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return copyDir(c, src, dst)
	}
	return copyFile(c, src, dst)
}

func copyDir(c *Context, src, dst string) error {
	// 자기 하위로 복사 금지
	srcAbs, _ := filepath.Abs(src)
	dstAbs, _ := filepath.Abs(dst)
	if isSubpath(dstAbs, srcAbs) {
		return fmt.Errorf("destination is inside source: %s -> %s", src, dst)
	}

	// 최상위 대상이 존재 & 파일이면 충돌 처리
	if st, err := os.Lstat(dst); err == nil && !st.IsDir() {
		action, err := resolveConflict(c, dst)
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
			act, err := resolveConflict(c, target)
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
		return copyOneFile(c, path, target, info.Mode().Perm())
	})
}

func copyFile(c *Context, src, dst string) error {
	// 대상이 디렉터리라면 파일명 붙임
	if di, err := os.Stat(dst); err == nil && di.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	}
	// 충돌 처리
	if exists(dst) {
		act, err := resolveConflict(c, dst)
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
	return copyOneFile(c, src, dst, srcFi.Mode().Perm())
}

func copyOneFile(c *Context, src, dst string, perm fs.FileMode) error {
	logger := c.Logger
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
			logger.Error("failed close file", "src", src)
		}
	}(in)

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		outErr := out.Close()
		if outErr != nil {
			logger.Error("failed close file", "dst", dst)
		}
	}(out)

	_, err = io.Copy(out, in)
	return err
}

func resolveConflict(c *Context, dst string) (string, error) {
	// 2-버튼 모달로 물어보기 (UI 스레드에서 생성)
	ch := make(chan string, 1)
	var dd dialog.Dialog

	show := func() {
		title := "already exists file"
		msg := widget.NewLabel(fmt.Sprintf("already exists file:\n%s\noverwrite?", dst))
		btnSkip := widget.NewButton("skip", func() { ch <- "skip"; dd.Hide() })
		btnOverwrite := widget.NewButton("overwrite", func() { ch <- "overwrite"; dd.Hide() })

		grid := container.NewGridWithColumns(2, btnSkip, btnOverwrite)
		content := container.NewVBox(msg, widget.NewSeparator(), grid)

		dd = dialog.NewCustomWithoutButtons(title, content, c.Window)
		dd.Show()
	}
	// UI 스레드에서 다이얼로그 띄우기
	fyne.Do(show)

	// 백그라운드(현재 고루틴)에서 사용자 선택 대기
	return <-ch, nil
}

func copyDirContents(c *Context, srcDir, dstDir string) error {
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
			if err := copyDir(c, s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(c, s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func handleCopy(c *Context, src string, dst string) error {
	logger := c.Logger
	absSrc, err := pathToAbs(c, src)
	if err != nil {
		logger.Error("failed path to abs", "src", src)
		return err
	}
	absDst, err := pathToAbs(c, dst)
	if err != nil {
		logger.Error("failed path to abs", "dst", dst)
		return err
	}

	err = copyEntry(c, absSrc, absDst)
	if err != nil {
		return err
	}

	c.RefreshSideBar()

	_, err = fmt.Fprintf(c.ConsoleBuf, "cp: %s to %s", absSrc, absDst)
	return err
}
