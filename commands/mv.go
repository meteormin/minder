package commands

import (
	"fmt"
	"os"
	"path/filepath"
)

var cmdMove = Cmd{
	Name: "mv",
	Args: []string{"<src>", "<dst>"},
	Exec: func(c *Context, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("mv: missing argument")
		}
		return handleMove(c, args[0], args[1])
	},
}

// moveEntry mv도 동일 정책
func moveEntry(c *Context, srcPattern, dst string) error {
	// 1) "aDir/." → 내용만 복사
	if dir, ok := asDotContents(srcPattern); ok {
		return copyDirContents(c, dir, dst)
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
			if err := moveDirContents(c, dir, dst); err != nil {
				return err
			}
			continue
		}
		if err = moveAny(c, s, dst); err != nil {
			return err
		}
	}
	return nil
}

// moveDirContents copyDirContents와 대칭
func moveDirContents(c *Context, srcDir, dstDir string) error {
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
		if err = moveAny(c, s, d); err != nil {
			return err
		}
	}
	return nil
}

func moveAny(c *Context, src, dst string) error {
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
	// 폴백: copyAny(+재귀) → remove
	if err := copyAny(c, src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func handleMove(c *Context, src string, dst string) error {
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

	err = moveEntry(c, absSrc, absDst)
	if err != nil {
		return err
	}

	c.RefreshSideBar()

	_, err = fmt.Fprintf(c.ConsoleBuf, "mv: %s to %s", absSrc, absDst)
	return err
}
