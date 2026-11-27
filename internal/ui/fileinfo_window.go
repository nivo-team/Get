package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"sheep-get/internal/window"
)

type fileInfoWindowView struct {
	app   *core.Scope
	state *appState
}

func (v *fileInfoWindowView) Show() {
	if v.app == nil {
		return
	}
	v.app.Post(func() {
		v.state.syncFileInfo()
		openFileInfoWindow(v.app, v.state)
	})
}

func (v *fileInfoWindowView) Hide() {
	if v.app == nil {
		return
	}
	v.app.Post(func() {
		v.state.closeFileInfoWindow()
	})
}

func (v *fileInfoWindowView) Focus() {
	if v.app == nil {
		return
	}
	v.app.Post(func() {
		v.state.syncFileInfo()
		openFileInfoWindow(v.app, v.state)
	})
}

func (v *fileInfoWindowView) Emit(_ string, _ any) {
	if v.app == nil {
		return
	}
	v.app.Post(func() {
		v.state.syncFileInfo()
	})
}

type fileInfoState struct {
	mu          sync.Mutex
	win         *desktop.Window
	activeItem  *window.FileInfoItem
	lastReqID   string
	editedName  bool
	editedDir   bool
	ver         *core.Signal[int]
	urlInput    *core.Signal[string]
	nameInput   *core.Signal[string]
	dirInput    *core.Signal[string]
	preDownload *core.Signal[bool]
	submitting  *core.Signal[bool]
	errorNotice *core.Signal[string]
}

func newFileInfoState(app *core.Scope) *fileInfoState {
	return &fileInfoState{
		ver:         core.NewSignal(app, 0),
		urlInput:    core.NewSignal(app, ""),
		nameInput:   core.NewSignal(app, ""),
		dirInput:    core.NewSignal(app, ""),
		preDownload: core.NewSignal(app, false),
		submitting:  core.NewSignal(app, false),
		errorNotice: core.NewSignal(app, ""),
	}
}

func (s *appState) syncFileInfo() {
	item, err := s.backend.GetActiveFileInfo()
	if err != nil {
		return
	}
	fi := s.fileInfo
	fi.mu.Lock()
	fi.activeItem = item
	if item == nil {
		fi.lastReqID = ""
		fi.editedName = false
		fi.editedDir = false
		fi.mu.Unlock()
		fi.ver.Update(func(v int) int { return v + 1 })
		return
	}
	isNewItem := fi.lastReqID != item.ID
	if isNewItem {
		fi.lastReqID = item.ID
		fi.editedName = false
		fi.editedDir = false
		fi.mu.Unlock()
		fi.urlInput.Set(item.URL)
		fi.nameInput.Set(item.Filename)
		fi.dirInput.Set(item.Directory)
		fi.preDownload.Set(item.PreDownload)
		fi.errorNotice.Set("")
	} else {
		editedName := fi.editedName
		editedDir := fi.editedDir
		fi.mu.Unlock()
		if item.URL != "" && strings.TrimSpace(fi.urlInput.Get()) == "" {
			fi.urlInput.Set(item.URL)
		}
		if !editedName && item.Filename != "" {
			fi.nameInput.Set(item.Filename)
		}
		if !editedDir && item.Directory != "" {
			fi.dirInput.Set(item.Directory)
		}
	}
	fi.ver.Update(func(v int) int { return v + 1 })
}

func (s *appState) closeFileInfoWindow() {
	fi := s.fileInfo
	fi.mu.Lock()
	win := fi.win
	fi.win = nil
	fi.mu.Unlock()
	if win != nil && !win.Closed() {
		win.Hide()
	}
}

func openFileInfoWindow(app *core.Scope, state *appState) *desktop.Window {
	win := desktop.OpenWindow(app, desktop.WindowOptions{
		Title:  "新建下载 - SheepGet",
		Key:    "fileinfo",
		Width:  480,
		Height: 380,
		Constraints: desktop.WindowConstraints{
			MinWidth:  420,
			MinHeight: 320,
			MaxHeight: 620,
		},
	}, func(cx *core.Scope) core.Node {
		return FileInfoWindowView(cx, state)
	})
	if win != nil {
		state.fileInfo.mu.Lock()
		state.fileInfo.win = win
		state.fileInfo.mu.Unlock()
		win.OnClosing(app, func(c *desktop.WindowClosing) {
			c.Cancel()
			_ = state.backend.CancelCurrentFileInfo()
		})
		win.Show()
		win.Focus()
	}
	return win
}

func (s *appState) currentFileInfoItem() *window.FileInfoItem {
	_ = s.fileInfo.ver.Get()
	s.fileInfo.mu.Lock()
	defer s.fileInfo.mu.Unlock()
	return s.fileInfo.activeItem
}

func (s *appState) submitFileInfo() {
	fi := s.fileInfo
	if fi.submitting.Get() {
		return
	}
	item := s.currentFileInfoItem()
	if item == nil {
		return
	}
	rawURL := strings.TrimSpace(fi.urlInput.Get())
	if rawURL == "" {
		fi.errorNotice.Set("请输入下载链接")
		return
	}
	sub := window.FileInfoSubmission{
		RequestID:   item.ID,
		URL:         rawURL,
		Filename:    strings.TrimSpace(fi.nameInput.Get()),
		Directory:   strings.TrimSpace(fi.dirInput.Get()),
		CategoryID:  item.CategoryID,
		MaxConn:     item.MaxConn,
		PreDownload: fi.preDownload.Get(),
	}
	if item.DuplicateDecision.Default != "" {
		sub.Action = item.DuplicateDecision.Default
	} else if len(item.DuplicateDecision.Options) > 0 {
		sub.Action = item.DuplicateDecision.Options[0]
	}

	fi.submitting.Set(true)
	fi.errorNotice.Set("")
	go func() {
		err := s.backend.SubmitFileInfo(sub)
		if s.app == nil {
			return
		}
		s.app.Post(func() {
			fi.submitting.Set(false)
			if err != nil {
				fi.errorNotice.Set(err.Error())
				return
			}
			s.syncFileInfo()
			s.refresh()
		})
	}()
}

// FileInfoWindowView renders the independent FileInfo confirmation dialog.
func FileInfoWindowView(cx *core.Scope, state *appState) core.Node {
	if desktop.IsAvailable(cx) {
		desktop.SetTitleBar(cx, desktop.TitleBarHidden, ui.HeaderHeight)
	}
	hideSystemWindowControls(cx)
	colors := ui.ColorsOf(cx)
	fi := state.fileInfo

	return insetWindowShell(cx,
		// Custom draggable header with queue pager & window controls
		core.View{
			Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOn},
			StyleFunc: func() core.Style {
				return core.Style{
					Direction:      core.Row,
					AlignItems:     core.AlignCenter,
					JustifyContent: core.JustifySpaceBetween,
					Height:         core.Pt(ui.HeaderHeight),
					Padding:        core.Symmetric(0, 14),
					Borders:        core.Edges{Bottom: 1},
					BorderColor:    colors.Line(),
				}
			},
			Children: []core.Node{
				core.View{
					Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
					Children: []core.Node{
						core.Text{
							Content: "新建下载",
							Style:   core.Style{FontSize: 13, FontWeight: core.FontWeightSemibold},
						},
						core.Show{
							When: func() bool {
								item := state.currentFileInfoItem()
								return item != nil && item.QueueTotal > 1
							},
							Then: func(cx *core.Scope) core.Node {
								return core.View{
									Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOff},
									Style:   core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4},
									Children: []core.Node{
										ui.Button{
											Icon:    lucide.ChevronLeft,
											Label:   "上一项",
											Variant: ui.ButtonGhost,
											Size:    ui.SizeSmall,
											Side:    headerButtonHeight,
											OnPress: func() {
												if item := state.currentFileInfoItem(); item != nil && item.QueueIndex > 0 {
													_, _ = state.backend.SwitchFileInfoActive(item.QueueIndex - 1)
													state.syncFileInfo()
												}
											},
										},
										ui.Badge{
											TextFunc: func() string {
												item := state.currentFileInfoItem()
												if item == nil {
													return ""
												}
												return fmt.Sprintf("%d/%d", item.QueueIndex+1, item.QueueTotal)
											},
											Variant: ui.BadgeNeutral,
										},
										ui.Button{
											Icon:    lucide.ChevronRight,
											Label:   "下一项",
											Variant: ui.ButtonGhost,
											Size:    ui.SizeSmall,
											Side:    headerButtonHeight,
											OnPress: func() {
												if item := state.currentFileInfoItem(); item != nil && item.QueueIndex+1 < item.QueueTotal {
													_, _ = state.backend.SwitchFileInfoActive(item.QueueIndex + 1)
													state.syncFileInfo()
												}
											},
										},
									},
								}
							},
						},
					},
				},
				core.View{
					Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOff},
					Style:   core.Style{Direction: core.Row, AlignItems: core.AlignCenter},
					Children: []core.Node{
						ui.Button{
							Icon:     lucide.X,
							Label:    "关闭窗口",
							Variant:  ui.ButtonGhost,
							Size:     ui.SizeSmall,
							Side:     headerButtonHeight,
							IconSize: 14,
							OnPress: func() {
								_ = state.backend.CancelCurrentFileInfo()
							},
						},
					},
				},
			},
		},
		// Form Body
		core.View{
			Style: core.Style{FlexGrow: 1, Padding: core.All(16), Gap: 12},
			Children: []core.Node{
				formField(cx, "下载链接", ui.Input{
					Value:       fi.urlInput.Get,
					OnChange:    fi.urlInput.Set,
					Placeholder: "https://example.com/file.zip",
					AutoFocus:   true,
					OnSubmit:    func(string) { state.submitFileInfo() },
				}),
				formField(cx, "文件名称", ui.Input{
					Value: fi.nameInput.Get,
					OnChange: func(v string) {
						fi.mu.Lock()
						fi.editedName = true
						fi.mu.Unlock()
						fi.nameInput.Set(v)
					},
					Placeholder: "自动探测或输入自定义文件名",
					OnSubmit:    func(string) { state.submitFileInfo() },
				}),
				formField(cx, "保存位置", ui.Input{
					Value: fi.dirInput.Get,
					OnChange: func(v string) {
						fi.mu.Lock()
						fi.editedDir = true
						fi.mu.Unlock()
						fi.dirInput.Set(v)
					},
					Placeholder: "默认下载目录",
					OnSubmit:    func(string) { state.submitFileInfo() },
				}),
				// File size / probe status & duplicate warning
				core.View{
					Style: core.Style{
						Direction:      core.Row,
						AlignItems:     core.AlignCenter,
						JustifyContent: core.JustifySpaceBetween,
						Gap:            8,
					},
					Children: []core.Node{
						core.Text{
							ContentFunc: func() string {
								item := state.currentFileInfoItem()
								if item == nil {
									return ""
								}
								if item.Probing {
									return "正在探测文件信息…"
								}
								if item.TotalBytes > 0 {
									return "文件大小：" + formatBytes(item.TotalBytes)
								}
								if item.URL != "" {
									return "文件大小：未知"
								}
								return "请输入或粘贴有效下载链接"
							},
							StyleFunc: func() core.Style {
								return ui.TypeCaption.Merge(core.Style{Color: colors.Muted()})
							},
						},
						core.Show{
							When: func() bool {
								item := state.currentFileInfoItem()
								return item != nil && item.FileConflict
							},
							Then: func(cx *core.Scope) core.Node {
								return ui.Badge{
									Text:    "目录中存在同名文件",
									Variant: ui.BadgeWarn,
								}
							},
						},
					},
				},
				core.Show{
					When: func() bool { return fi.errorNotice.Get() != "" },
					Then: func(cx *core.Scope) core.Node {
						return core.Text{
							ContentFunc: fi.errorNotice.Get,
							StyleFunc: func() core.Style {
								return ui.TypeCaption.Merge(core.Style{Color: colors.Danger()})
							},
						}
					},
				},
			},
		},
		// Footer Actions
		core.View{
			StyleFunc: func() core.Style {
				return core.Style{
					Direction:      core.Row,
					AlignItems:     core.AlignCenter,
					JustifyContent: core.JustifySpaceBetween,
					Padding:        core.Symmetric(10, 16),
					Borders:        core.Edges{Top: 1},
					BorderColor:    colors.Line(),
				}
			},
			Children: []core.Node{
				core.View{
					Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
					Children: []core.Node{
						ui.Checkbox{
							Checked:  fi.preDownload.Get,
							OnChange: fi.preDownload.Set,
							Label:    "提前下载",
						},
					},
				},
				core.View{
					Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
					Children: []core.Node{
						ui.Button{
							Text:    "取消",
							Variant: ui.ButtonGhost,
							Size:    ui.SizeSmall,
							Style:   core.Style{Height: core.Pt(headerButtonHeight)},
							OnPress: func() {
								_ = state.backend.CancelCurrentFileInfo()
							},
						},
						ui.Button{
							Text:    "开始下载",
							Variant: ui.ButtonPrimary,
							Size:    ui.SizeSmall,
							Style:   core.Style{Height: core.Pt(headerButtonHeight)},
							Busy:    fi.submitting.Get,
							OnPress: state.submitFileInfo,
						},
					},
				},
			},
		},
	)
}

func formField(cx *core.Scope, label string, input core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{
		Style: core.Style{Gap: 4},
		Children: []core.Node{
			core.Text{
				Content: label,
				StyleFunc: func() core.Style {
					return ui.TypeCaption.Merge(core.Style{Color: colors.Muted(), FontWeight: core.FontWeightMedium})
				},
			},
			input,
		},
	}
}
