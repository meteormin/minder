package components

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/meteormin/minder"
)

type FileTree struct {
	*widget.Tree
	rootDir        string
	onFileSelected func(c *minder.Context, s string)
}

// NewFileTree 지정한 루트로
func NewFileTree(c *minder.Context, onFileSelected func(c *minder.Context, s string)) (*FileTree, error) {
	logger, _ := c.Get("logger").(*slog.Logger)
	root, _ := c.Get("filePath").(string)
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		root = wd
	}

	ft := &FileTree{
		rootDir:        root,
		onFileSelected: onFileSelected,
	}

	// ✅ 콜백 생성자 사용
	ft.Tree = widget.NewTree(
		ft.childUIDs,
		ft.isBranch,
		ft.createItem,
		ft.updateItem,
	)

	// ✅ 루트 지정
	ft.Tree.Root = ft.rootDir
	ft.Tree.OpenBranch(ft.rootDir)

	// ✅ 폴더 토글
	ft.Tree.OnSelected = func(uid string) {
		if uid == "" {
			return
		}
		// 루트 밖 선택 방지
		if rel, err := filepath.Rel(ft.rootDir, uid); err != nil || strings.HasPrefix(rel, "..") {
			logger.Error("skip selection out of root", "uid", uid, "err", err)
			return
		}

		info, err := os.Lstat(uid)
		if err != nil {
			logger.Error("stat error", "uid", uid, "err", err)
			return
		}

		isDir := info.IsDir()
		// 심볼릭 링크가 디렉터리를 가리키면 디렉터리처럼 취급
		if !isDir && info.Mode()&os.ModeSymlink != 0 {
			if ti, err := os.Stat(uid); err == nil && ti.IsDir() {
				isDir = true
			}
		}

		if isDir {
			if ft.Tree.IsBranchOpen(uid) {
				ft.Tree.CloseBranch(uid)
			} else {
				ft.Tree.OpenBranch(uid)
			}
			ft.Tree.UnselectAll()
			return
		}

		if ft.onFileSelected != nil {
			ft.onFileSelected(c, uid)
		}
	}

	// 화살표(▶)로 폴더가 열릴 때도 동일 로직 수행
	ft.Tree.OnBranchOpened = func(uid string) {
		ft.Tree.RefreshItem(uid)
	}

	ft.Tree.OnBranchClosed = func(uid string) {
		ft.Tree.RefreshItem(uid)
	}

	return ft, nil
}

func (ft *FileTree) SetRootDir(root string) {
	if root == "" {
		return
	}
	ft.rootDir = root
	ft.Tree.Root = root
	ft.Tree.OpenBranch(root)
	ft.Tree.Refresh()
}

func (ft *FileTree) childUIDs(uid string) []string {
	path := ft.rootDir
	if uid != "" {
		path = uid
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		log.Printf("readDir error: %s: %v", path, err)
		return nil
	}

	type item struct {
		name string
		path string
		dir  bool
	}
	var tmp []item
	for _, e := range entries {
		name := e.Name()
		// 숨김 파일/폴더 제외
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(path, name)
		isDir := e.IsDir()

		// 링크가 디렉터리를 가리키면 디렉터리로 간주
		if !isDir {
			if info, err := os.Lstat(full); err == nil && info.Mode()&os.ModeSymlink != 0 {
				if ti, err := os.Stat(full); err == nil && ti.IsDir() {
					isDir = true
				}
			}
		}
		tmp = append(tmp, item{name: name, path: full, dir: isDir})
	}

	// 폴더 우선 + 이름 오름차순
	sort.Slice(tmp, func(i, j int) bool {
		if tmp[i].dir != tmp[j].dir {
			return tmp[i].dir
		}
		return strings.ToLower(tmp[i].name) < strings.ToLower(tmp[j].name)
	})

	var uids []string
	for _, it := range tmp {
		uids = append(uids, it.path)
	}
	return uids
}

func (ft *FileTree) isBranch(uid string) bool {
	if uid == "" {
		return true
	}
	if info, err := os.Lstat(uid); err == nil {
		if info.IsDir() {
			return true
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if ti, err := os.Stat(uid); err == nil && ti.IsDir() {
				return true
			}
		}
	}
	return false
}

func (ft *FileTree) createItem(branch bool) fyne.CanvasObject {
	icon := widget.NewIcon(theme.FileIcon())
	if branch {
		icon = widget.NewIcon(theme.FolderIcon())
	}
	lbl := widget.NewLabel("")
	lbl.Truncation = fyne.TextTruncateClip
	return container.NewHBox(icon, lbl)
}

func (ft *FileTree) updateItem(uid string, branch bool, obj fyne.CanvasObject) {
	if uid == "" {
		return
	}
	box := obj.(*fyne.Container)
	// [0]=Icon, [1]=Label 가정
	if len(box.Objects) >= 2 {
		if ic, ok := box.Objects[0].(*widget.Icon); ok {
			if branch {
				ic.SetResource(theme.FolderIcon())
			} else {
				ic.SetResource(theme.FileIcon())
			}
		}
		if lb, ok := box.Objects[1].(*widget.Label); ok {
			lb.SetText(filepath.Base(uid))
		}
	}
}

type FileSelector struct {
}

func Pathfinder(c *minder.Context, onFileSelected func(c *minder.Context, s string)) fyne.CanvasObject {
	logger, _ := c.Get("logger").(*slog.Logger)
	currentDir := c.Get("filePath").(string)
	label := widget.NewLabel(currentDir)
	fTree, err := NewFileTree(c, onFileSelected)
	if err != nil {
		logger.Error("failed new file tree", "err", err)
		panic(err)
	}
	return container.NewBorder(label, nil, nil, nil, fTree)
}
