package components

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type FileTree struct {
	*widget.Tree
	rootDir    binding.String
	showHidden binding.Bool
	win        fyne.Window
	onSelected func(string)

	open   map[string]struct{} // ★ 현재 열려 있는 브랜치 집합
	unsubs []func()            // 바인딩 리스너 해제
}

type FileTreeConfig struct {
	Window     fyne.Window
	RootDir    binding.String
	ShowHidden binding.Bool
	OnSelected func(uid string)
}

func NewFileTreeWithData(cfg FileTreeConfig) (*FileTree, error) {
	ft := &FileTree{
		rootDir:    cfg.RootDir,
		showHidden: cfg.ShowHidden,
		win:        cfg.Window,
		onSelected: cfg.OnSelected,
		open:       map[string]struct{}{}, // ★
	}

	// 콜백 기반 Tree
	ft.Tree = widget.NewTree(
		ft.childUIDs,
		ft.isBranch,
		ft.createItem,
		ft.updateItem,
	)

	// 루트 설정
	root, err := ft.rootDir.Get()
	if err != nil {
		return nil, err
	}

	ft.Tree.Root = root
	ft.open[root] = struct{}{} // ★ 루트를 열린 것으로 간주
	ft.Tree.OpenBranch(root)
	ft.Tree.OnSelected = func(uid string) {
		ft.Tree.UnselectAll()
		if uid == "" {
			return
		}

		// 루트 경계 체크
		curRoot, _ := ft.rootDir.Get()
		if rel, err := filepath.Rel(curRoot, uid); err != nil || strings.HasPrefix(rel, "..") {
			dialog.ShowError(err, ft.win)
			return
		}

		info, err := os.Lstat(uid)
		if err != nil {
			dialog.ShowError(err, ft.win)
			return
		}

		isDir := info.IsDir()
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
		} else if ft.onSelected != nil {
			ft.onSelected(uid)
		}

		ft.win.Clipboard().SetContent(uid)
	}

	// 브랜치 토글 시 아이콘 새로고침
	ft.Tree.OnBranchOpened = func(uid string) {
		ft.open[uid] = struct{}{} // ★
		ft.Tree.RefreshItem(uid)  // 아이콘/자식 목록 갱신
	}
	ft.Tree.OnBranchClosed = func(uid string) {
		delete(ft.open, uid) // ★
		ft.Tree.RefreshItem(uid)
	}

	// ✨ 바인딩 연동: RootDir/ShowHidden 값이 바뀌면 자동 갱신
	ft.bind()

	return ft, nil
}

func (ft *FileTree) Refresh() {
	ft.Tree.Refresh()
	ft.RefreshItem(ft.Root)
}

// SetRootDir 외부에서 문자열로 루트 바꾸고 싶을 때도 binding으로 통일
func (ft *FileTree) SetRootDir(root string) {
	if root == "" {
		return
	}
	_ = ft.rootDir.Set(root) // 리스너에서 UI 반영
}

// --- 내부 구현 ---

// NewFileTreeWithData() 내부 마지막에 한 번만 호출
func (ft *FileTree) bind() {
	// 중복 구독 방지: 기존 리스너 해제
	for _, u := range ft.unsubs {
		u()
	}
	ft.unsubs = nil

	// 1) RootDir 변경 시: 루트 교체 + 브랜치 열기 + Refresh
	rdL := binding.NewDataListener(func() {
		root, _ := ft.rootDir.Get()
		fyne.Do(func() {
			// 기존 열린 브랜치 닫기(시각적 일관성 확보 – 선택)
			for uid := range ft.open {
				ft.Tree.CloseBranch(uid)
			}
			ft.open = map[string]struct{}{} // ★ 리셋

			ft.Tree.Root = root
			ft.open[root] = struct{}{} // ★ 새 루트 등록
			ft.Tree.OpenBranch(root)

			// 루트부터 다시 자식 계산
			ft.Tree.RefreshItem(root) // ★ 중요
			ft.Tree.Refresh()
		})
	})
	ft.rootDir.AddListener(rdL)
	ft.unsubs = append(ft.unsubs, func() { ft.rootDir.RemoveListener(rdL) })

	// 2) ShowHidden 변경 시: 목록만 새로고침
	shL := binding.NewDataListener(func() {
		fyne.Do(func() {
			for uid := range ft.open { // ★ 열린 부모마다
				ft.Tree.RefreshItem(uid) //   자식 목록 다시 계산
			}
			ft.Tree.Refresh() // 화면 재도색
		})
	})
	ft.showHidden.AddListener(shL)
	ft.unsubs = append(ft.unsubs, func() { ft.showHidden.RemoveListener(shL) })
}

func (ft *FileTree) childUIDs(uid string) []string {
	root, _ := ft.rootDir.Get()
	path := root
	if uid != "" {
		path = uid
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		fyne.LogError("read dir failed "+path, err)
		return nil
	}

	showHidden, _ := ft.showHidden.Get()

	type item struct {
		name string
		path string
		dir  bool
	}
	var tmp []item
	for _, e := range entries {
		name := e.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(path, name)
		isDir := e.IsDir()
		// 심볼릭 링크가 디렉터리를 가리키면 디렉터리로 간주
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

	uids := make([]string, 0, len(tmp))
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

// Destroy 필요 시 호출: 바인딩 리스너 해제
func (ft *FileTree) Destroy() {
	for _, u := range ft.unsubs {
		u()
	}
	ft.unsubs = nil
}

type PathfinderState struct {
	CurrentDir binding.String
	ShowHidden binding.Bool
}

type Pathfinder struct {
	State     PathfinderState
	Container *fyne.Container
}

type PathfinderConfig struct {
	FileTreeConfig
	Logger *slog.Logger
}

func NewPathfinder(cfg PathfinderConfig) *Pathfinder {
	label := widget.NewLabelWithData(cfg.RootDir)
	fTree, err := NewFileTreeWithData(cfg.FileTreeConfig)
	if err != nil {
		cfg.Logger.Error("failed new file tree", "err", err)
		panic(err)
	}

	check := widget.NewCheck("hidden", func(b bool) {
		err = cfg.ShowHidden.Set(b)
		if err != nil {
			cfg.Logger.Error("failed set show hidden", "err", err)
		}
	})

	topBox := container.NewVBox(label, check)

	c := container.NewBorder(topBox, nil, nil, nil, fTree)

	return &Pathfinder{
		State: PathfinderState{
			CurrentDir: cfg.RootDir,
			ShowHidden: cfg.ShowHidden,
		},
		Container: c,
	}
}
