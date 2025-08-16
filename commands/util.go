package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/meteormin/minder"
)

type conflictMode int

const (
	modeAsk conflictMode = iota
	modeOverwriteAll
	modeSkipAll
)

func pathToAbs(c *minder.Context, dst string) (string, error) {
	var fp string
	if filepath.IsAbs(dst) {
		if _, err := os.Stat(dst); err != nil {
			return "", err
		}
		fp = filepath.Dir(dst)
	}

	currentDir, _ := c.Get("filePath").(string)
	s, err := os.Lstat(currentDir)
	if err != nil {
		return "", err
	}

	if s.IsDir() {
		fp = filepath.Join(currentDir, dst)
	} else {
		fp = filepath.Join(filepath.Dir(currentDir), dst)
	}

	return fp, nil
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

func isSubpath(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." && rel != "." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func isCrossDevice(err error) bool {
	var le *os.LinkError
	if errors.As(err, &le) {
		return errors.Is(le.Err, syscall.EXDEV)
	}
	return errors.Is(err, syscall.EXDEV)
}

func expandPattern(p string) ([]string, error) {
	if !hasGlob(p) {
		return []string{p}, nil
	}
	matches, err := filepath.Glob(p)
	if err != nil {
		return nil, err
	}
	basePat := filepath.Base(p)
	if strings.HasPrefix(basePat, ".") {
		// ".*" 류: "." ".." 제거
		out := matches[:0]
		for _, m := range matches {
			b := filepath.Base(m)
			if b == "." || b == ".." {
				continue
			}
			out = append(out, m)
		}
		matches = out
	} else {
		// "*" 류: dotfile 제외
		out := matches[:0]
		for _, m := range matches {
			if strings.HasPrefix(filepath.Base(m), ".") {
				continue
			}
			out = append(out, m)
		}
		matches = out
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no matches for %q", p)
	}
	return matches, nil
}

// hasGlob: 글롭 문자가 있는지
func hasGlob(p string) bool { return strings.ContainsAny(p, "*?[") }

// asDotContents: "aDir/." → ("aDir", true)
func asDotContents(p string) (string, bool) {
	clean := filepath.Clean(p)
	if filepath.Base(clean) == "." {
		return filepath.Dir(clean), true
	}
	return "", false
}

func shouldFallbackRename(err error) bool {
	// Windows 드라이브 간 이동 등 다양한 에러 → 폴백 권장
	return runtime.GOOS == "windows"
}
