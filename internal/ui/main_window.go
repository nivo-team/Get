package ui

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"sheep-get/internal/clipboard"
	"sheep-get/internal/config"
	"sheep-get/internal/server"
	"sheep-get/internal/task"
	"sheep-get/internal/window"
)

// Backend abstracts the Go application core operations needed by the native UI.
type Backend interface {
	ListTasks() ([]*task.Task, error)
	PauseTask(id string) error
	ResumeTask(id string) error
	RetryTask(id string) error
	DeleteTask(id string, deleteDiskFile bool) error
	OpenFile(path string) error
	OpenFolder(path string) error
	OpenNewDownloadWithClipboard(text string) error
	GetActiveFileInfo() (*window.FileInfoItem, error)
	SubmitFileInfo(sub window.FileInfoSubmission) error
	CancelCurrentFileInfo() error
	SwitchFileInfoActive(index int) (*window.FileInfoItem, error)
	SetFileInfoWindowView(view window.WindowView)
	GetServerStatus() server.Status
	GetSettings() config.Settings
	UpdateSettings(s config.Settings) (config.Settings, error)
	OpenExtensionFolder() error
	OpenSkillDirectory() error
	AttachClipboard(reader clipboard.Reader)
	SetOnTaskChanged(fn func())
	SetOnShowProgress(fn func(taskID string))
}

type filterDef struct {
	id    TaskFilter
	title string
	icon  func() *core.SVG
}

var navFilters = []filterDef{
	{FilterAll, "全部任务", lucide.Layers},
	{FilterDownloading, "下载中", lucide.CircleArrowDown},
	{FilterCompleted, "已完成", lucide.CircleCheck},
	{FilterPaused, "已暂停", lucide.CirclePause},
	{FilterFailed, "失败", lucide.TriangleAlert},
}

type appState struct {
	app          *core.Scope
	backend      Backend
	fileInfo     *fileInfoState
	settings     *settingsState
	tasksMu      sync.RWMutex
	tasksList    []*task.Task
	trackedIDs   map[string]struct{}
	dismissedIDs map[string]struct{}
	taskVer      *core.Signal[int]
	filter       *core.Signal[TaskFilter]
	sidebarOpen  *core.Signal[bool]
	statusNotice *core.Signal[string]
	serverStatus *core.Signal[server.Status]
}

// Run starts the NSGo desktop application loop, opens the main window, and registers the system tray.
func Run(backend Backend, trayPNG []byte) {
	initialSettings := backend.GetSettings()
	appearanceOpt := appearanceOption(initialSettings.Appearance.Theme)
	nsgo.Run(func(app *core.Scope) {
		desktop.SetShutdownMode(app, desktop.ShutdownOnExplicit)

		state := &appState{
			app:          app,
			backend:      backend,
			fileInfo:     newFileInfoState(app),
			settings:     newSettingsState(app, initialSettings),
			trackedIDs:   make(map[string]struct{}),
			dismissedIDs: make(map[string]struct{}),
			taskVer:      core.NewSignal(app, 0),
			filter:       core.NewSignal(app, FilterAll),
			sidebarOpen:  core.NewSignal(app, true),
			statusNotice: core.NewSignal(app, ""),
			serverStatus: core.NewSignal(app, backend.GetServerStatus()),
		}
		state.refresh()

		backend.SetFileInfoWindowView(&fileInfoWindowView{app: app, state: state})
		backend.SetOnTaskChanged(func() {
			app.Post(func() {
				state.refresh()
			})
		})
		backend.SetOnShowProgress(func(taskID string) {
			app.Post(func() {
				state.showProgressWindow(taskID)
			})
		})
		backend.AttachClipboard(newScopeClipboard(app))

		openMain := func() *desktop.Window {
			win := desktop.OpenWindow(app, desktop.WindowOptions{
				Title:  "SheepGet",
				Key:    "main",
				Width:  880,
				Height: 560,
				Constraints: desktop.WindowConstraints{
					MinWidth:  720,
					MinHeight: 440,
				},
			}, func(cx *core.Scope) core.Node {
				return MainWindowView(cx, state)
			})
			if win != nil {
				desktop.SetMainWindow(app, win)
				win.OnClosing(app, func(c *desktop.WindowClosing) {
					c.Cancel()
					win.Hide()
				})
			}
			return win
		}

		openMain()

		desktop.NewTray(app, desktop.TrayOptions{
			Icon:    trayPNG,
			Tooltip: "SheepGet",
			OnClick: func() {
				if win := openMain(); win != nil {
					win.Show()
					win.Focus()
				}
			},
			Menu: func() []desktop.MenuItem {
				return []desktop.MenuItem{
					{
						Label: "显示主窗口",
						OnSelect: func() {
							if win := openMain(); win != nil {
								win.Show()
								win.Focus()
							}
						},
					},
					{
						Label: "设置…",
						OnSelect: func() {
							state.filter.Set(FilterSettings)
							if win := openMain(); win != nil {
								win.Show()
								win.Focus()
							}
						},
					},
					{Separator: true},
					{
						Label: "退出 SheepGet",
						OnSelect: func() {
							app.Quit()
						},
					},
				}
			},
		})
	}, appearanceOpt)
}

func appearanceOption(theme config.ThemeMode) nsgo.Option {
	switch theme {
	case config.ThemeLight:
		return nsgo.Appearance(core.AppearanceLight)
	case config.ThemeDark:
		return nsgo.Appearance(core.AppearanceDark)
	default:
		return nsgo.Appearance(core.AppearanceSystem)
	}
}

func (s *appState) refresh() {
	list, err := s.backend.ListTasks()
	if err == nil {
		s.tasksMu.Lock()
		s.tasksList = list
		for _, t := range list {
			if t == nil {
				continue
			}
			if t.Status == task.StatusDownloading || t.Status == task.StatusQueued || t.Status == task.StatusProcessing {
				s.trackedIDs[t.ID] = struct{}{}
				delete(s.dismissedIDs, t.ID)
			}
		}
		s.tasksMu.Unlock()
		s.taskVer.Update(func(v int) int { return v + 1 })
	}
	s.serverStatus.Set(s.backend.GetServerStatus())
}

func (s *appState) filteredTasks() []*task.Task {
	_ = s.taskVer.Get()
	s.tasksMu.RLock()
	list := s.tasksList
	s.tasksMu.RUnlock()
	return filterTasks(list, s.filter.Get())
}

func (s *appState) progressTasks() (completed, active []*task.Task) {
	_ = s.taskVer.Get()
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()
	return progressWindowTasks(s.tasksList, s.trackedIDs, s.dismissedIDs)
}

func (s *appState) showProgressWindow(taskID string) {
	if taskID != "" {
		s.tasksMu.Lock()
		s.trackedIDs[taskID] = struct{}{}
		delete(s.dismissedIDs, taskID)
		s.tasksMu.Unlock()
	}
	s.refresh()
	if s.app != nil {
		openProgressWindow(s.app, s)
	}
}

func (s *appState) dismissTask(taskID string) {
	s.tasksMu.Lock()
	s.dismissedIDs[taskID] = struct{}{}
	s.tasksMu.Unlock()
	s.taskVer.Update(func(v int) int { return v + 1 })
}

func (s *appState) dismissAllCompleted() {
	s.tasksMu.Lock()
	for _, t := range s.tasksList {
		if t != nil && t.Status == task.StatusCompleted {
			s.dismissedIDs[t.ID] = struct{}{}
		}
	}
	s.tasksMu.Unlock()
	s.taskVer.Update(func(v int) int { return v + 1 })
}

func (s *appState) currentFilterTitle() string {
	cur := s.filter.Get()
	if cur == FilterSettings {
		return "设置"
	}
	for _, f := range navFilters {
		if f.id == cur {
			return f.title
		}
	}
	return "全部任务"
}

func (s *appState) openNewDownload() {
	// 剪贴板是异步读取的，不能在界面线程上等：先读到手边的文本，再交给后端判定能否当链接用。
	s.app.ReadClipboard(func(text string) {
		if err := s.backend.OpenNewDownloadWithClipboard(text); err != nil {
			s.statusNotice.Set("打开新建下载失败：" + err.Error())
			return
		}
		s.statusNotice.Set("")
	})
}

// MainWindowView builds the declarative NSGo view tree for the main window.
func MainWindowView(cx *core.Scope, state *appState) core.Node {
	if desktop.IsAvailable(cx) {
		desktop.SetTitleBar(cx, desktop.TitleBarHidden, ui.HeaderHeight)
	}
	hideSystemWindowControls(cx)
	colors := ui.ColorsOf(cx)
	items := make([]ui.SidebarItem, len(navFilters))
	for i, f := range navFilters {
		items[i] = ui.SidebarItem{
			ID:     string(f.id),
			Text:   f.title,
			Icon:   f.icon,
			TestID: "nav-" + string(f.id),
		}
	}

	const sidebarWidth core.Points = 216
	const sidebarInset core.Points = 5
	shown, reveal := sidebarSlideFrames(cx, state.sidebarOpen.Get)

	sidebarPane := core.View{
		StyleFunc: func() core.Style {
			st := core.Style{
				Width:      core.Pt(sidebarWidth * reveal()),
				FlexShrink: 0,
				MinHeight:  core.Pt(0),
				Overflow:   core.Hidden,
			}
			if !shown.Get() {
				st.Visibility = core.VisibilityHidden
			}
			return st
		},
		Children: []core.Node{
			// Inner fixed-width container so sidebar text never wraps/squishes during slide animation
			core.View{
				Style: core.Style{
					Width:     core.Pt(sidebarWidth),
					FlexGrow:  1,
					MinHeight: core.Pt(0),
				},
				Children: []core.Node{
					// Sidebar top drag area with collapse toggle
					core.View{
						Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOn},
						Style: core.Style{
							Direction:  core.Row,
							AlignItems: core.AlignCenter,
							Height:     core.Pt(ui.HeaderHeight),
							Padding:    core.Symmetric(0, 10),
						},
						Children: []core.Node{
							core.View{
								Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOff},
								Children: []core.Node{
									ui.Button{
										Icon:    lucide.PanelLeft,
										Label:   "收起侧边栏",
										Variant: ui.ButtonGhost,
										Size:    ui.SizeSmall,
										Side:    headerButtonHeight,
										OnPress: func() { state.sidebarOpen.Set(false) },
									},
								},
							},
						},
					},
					// Sidebar nav + status footer
					core.View{
						Style: core.Style{
							FlexGrow:       1,
							FlexShrink:     1,
							MinHeight:      core.Pt(0),
							JustifyContent: core.JustifySpaceBetween,
						},
						Children: []core.Node{
							ui.Sidebar{
								TestID: "main-sidebar",
								Selected: func() string {
									return string(state.filter.Get())
								},
								OnSelect: func(id string) {
									state.filter.Set(TaskFilter(id))
								},
								Sections: []ui.SidebarSection{
									{Title: "下载管理", Items: items},
									{
										Title: "系统",
										Items: []ui.SidebarItem{
											{
												ID:     string(FilterSettings),
												Text:   "设置",
												Icon:   lucide.Settings,
												TestID: "nav-settings",
											},
										},
									},
								},
							},
							core.View{
								Style: core.Style{
									Padding: core.Symmetric(10, 14),
									Gap:     4,
								},
								Children: []core.Node{
									core.Text{
										ContentFunc: func() string {
											st := state.serverStatus.Get()
											if st.Running {
												return fmt.Sprintf("服务就绪 · 端口 %d", st.Port)
											}
											return "本地服务未启动"
										},
										StyleFunc: func() core.Style {
											return ui.TypeCaption.Merge(core.Style{Color: colors.Muted()})
										},
									},
									core.Text{
										ContentFunc: func() string {
											st := state.serverStatus.Get()
											return fmt.Sprintf("浏览器扩展连接：%d", st.ConnectedCount)
										},
										StyleFunc: func() core.Style {
											return ui.TypeCaption.Merge(core.Style{Color: colors.Faint()})
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	contentCard := core.View{
		StyleFunc: func() core.Style {
			return core.Style{
				FlexGrow:     1,
				FlexShrink:   1,
				MinHeight:    core.Pt(0),
				Background:   colors.Background(),
				Color:        colors.Text(),
				CornerRadius: 10,
				BorderCurve:  core.BorderCurveContinuous,
				BorderWidth:  1,
				BorderColor:  colors.Line(),
				Overflow:     core.Hidden,
			}
		},
		Children: []core.Node{
			// Content header (draggable)
			core.View{
				Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOn},
				StyleFunc: func() core.Style {
					return core.Style{
						Direction:      core.Row,
						AlignItems:     core.AlignCenter,
						JustifyContent: core.JustifySpaceBetween,
						Height:         core.Pt(ui.HeaderHeight),
						Padding:        core.Symmetric(0, 14),
						Gap:            12,
						Borders:        core.Edges{Bottom: 1},
						BorderColor:    colors.Line(),
					}
				},
				Children: []core.Node{
					core.View{
						Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
						Children: []core.Node{
							core.Show{
								When: func() bool { return !state.sidebarOpen.Get() },
								Then: func(cx *core.Scope) core.Node {
									return core.View{
										Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOff},
										Children: []core.Node{
											ui.Button{
												Icon:    lucide.PanelLeft,
												Label:   "展开侧边栏",
												Variant: ui.ButtonGhost,
												Size:    ui.SizeSmall,
												Side:    headerButtonHeight,
												OnPress: func() { state.sidebarOpen.Set(true) },
											},
										},
									}
								},
							},
							core.Text{
								ContentFunc: state.currentFilterTitle,
								Style:       core.Style{FontSize: 14, FontWeight: core.FontWeightSemibold},
							},
							core.Show{
								When: func() bool { return state.filter.Get() != FilterSettings },
								Then: func(cx *core.Scope) core.Node {
									return ui.Badge{
										TextFunc: func() string {
											return fmt.Sprintf("%d", len(state.filteredTasks()))
										},
										Variant: ui.BadgeNeutral,
									}
								},
							},
						},
					},
					core.View{
						Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOff},
						Style:   core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6},
						Children: []core.Node{
							ui.Button{
								Text:    "新建下载",
								Icon:    lucide.Plus,
								Variant: ui.ButtonPrimary,
								Size:    ui.SizeSmall,
								Style:   core.Style{Height: core.Pt(headerButtonHeight)},
								OnPress: state.openNewDownload,
							},
							ui.Button{
								Text:    "进度窗口",
								Icon:    lucide.Activity,
								Variant: ui.ButtonSecondary,
								Size:    ui.SizeSmall,
								Style:   core.Style{Height: core.Pt(headerButtonHeight)},
								OnPress: func() {
									state.showProgressWindow("")
								},
							},
							ui.Button{
								Icon:    lucide.RefreshCw,
								Label:   "刷新列表",
								Variant: ui.ButtonGhost,
								Size:    ui.SizeSmall,
								Side:    headerButtonHeight,
								OnPress: state.refresh,
							},
							windowControlButtons(cx, true),
						},
					},
				},
			},
			// Content body
			core.Show{
				When: func() bool { return state.statusNotice.Get() != "" },
				Then: func(cx *core.Scope) core.Node {
					return core.View{
						Style: core.Style{Padding: core.Symmetric(6, 16)},
						Children: []core.Node{
							core.Text{
								ContentFunc: state.statusNotice.Get,
								StyleFunc: func() core.Style {
									return ui.TypeCaption.Merge(core.Style{Color: colors.Danger()})
								},
							},
						},
					}
				},
			},
			core.Show{
				When: func() bool { return state.filter.Get() == FilterSettings },
				Then: func(cx *core.Scope) core.Node {
					return SettingsView(cx, state)
				},
				Else: func(cx *core.Scope) core.Node {
					return core.Show{
						When: func() bool { return len(state.filteredTasks()) == 0 },
						Then: func(cx *core.Scope) core.Node {
							return emptyTaskView(cx)
						},
						Else: func(cx *core.Scope) core.Node {
							return ui.ScrollView{
								Style:        core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: core.Pt(0)},
								ContentStyle: core.Style{Padding: core.All(16), Gap: 8},
								Children: []core.Node{
									core.Each(
										state.filteredTasks,
										func(t *task.Task) string { return t.ID },
										func(cx *core.Scope, item func() *task.Task, _ func() int) core.Node {
											return taskRow(cx, state, item)
										},
									),
								},
							}
						},
					}
				},
			},
		},
	}

	return core.View{
		TestID: "main-shell",
		StyleFunc: func() core.Style {
			r := reveal()
			return core.Style{
				FlexGrow:   1,
				FlexShrink: 1,
				MinHeight:  core.Pt(0),
				Direction:  core.Row,
				Padding:    core.All(sidebarInset * r),
				Gap:        sidebarInset * r,
				Background: ui.Chrome(cx),
				Color:      colors.Text(),
			}
		},
		Children: []core.Node{
			sidebarPane,
			contentCard,
		},
	}
}

func sidebarSlideFrames(cx *core.Scope, open func() bool) (*core.Signal[bool], func() core.Points) {
	const duration = 220 * time.Millisecond
	env := cx.Env()
	was := core.Untracked(cx, open)
	var initReveal core.Points
	if was {
		initReveal = 1
	}
	shown := core.NewSignal(cx, was)
	reveal := core.NewSignal(cx, initReveal)
	var stop func()

	cx.Effect(func() {
		o := open()
		if o == was {
			return
		}
		was = o
		cx.Untrack(func() {
			if stop != nil {
				stop()
				stop = nil
			}
			var to core.Points
			if o {
				to = 1
				shown.Set(true)
			}
			from := reveal.Get()
			if env.ReduceMotion() {
				reveal.Set(to)
				shown.Set(o)
				return
			}
			stop = cx.Animate(duration, core.EaseInOut, func(p float64) {
				reveal.Set(from + (to-from)*core.Points(p))
				if p == 1 && !o {
					shown.Set(false)
				}
			})
		})
	})
	return shown, reveal.Get
}

func emptyTaskView(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{
		Style: core.Style{
			FlexGrow:       1,
			AlignItems:     core.AlignCenter,
			JustifyContent: core.JustifyCenter,
			Gap:            10,
			Padding:        core.All(32),
		},
		Children: []core.Node{
			core.Icon{
				SVG: lucide.Inbox(),
				StyleFunc: func() core.Style {
					return core.Style{Width: core.Pt(36), Height: core.Pt(36), Color: colors.Faint()}
				},
			},
			core.Text{
				Content: "暂无下载任务",
				StyleFunc: func() core.Style {
					return ui.TypeBody.Merge(core.Style{FontWeight: core.FontWeightMedium, Color: colors.Text()})
				},
			},
			core.Text{
				Content: "点击右上角「新建下载」或通过浏览器扩展接管下载",
				StyleFunc: func() core.Style {
					return ui.TypeCaption.Merge(core.Style{Color: colors.Muted()})
				},
			},
		},
	}
}

func taskRow(cx *core.Scope, state *appState, item func() *task.Task) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{
		StyleFunc: func() core.Style {
			return core.Style{
				Padding:      core.Symmetric(10, 14),
				Gap:          8,
				CornerRadius: ui.RadiusControl,
				BorderWidth:  1,
				BorderColor:  colors.Line(),
				Background:   colors.Surface(),
			}
		},
		Children: []core.Node{
			core.View{
				Style: core.Style{
					Direction:      core.Row,
					AlignItems:     core.AlignCenter,
					JustifyContent: core.JustifySpaceBetween,
					Gap:            12,
				},
				Children: []core.Node{
					core.View{
						Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 2},
						Children: []core.Node{
							core.Text{
								ContentFunc: func() string {
									t := item()
									if t == nil || t.Filename == "" {
										return "未命名任务"
									}
									return t.Filename
								},
								StyleFunc: func() core.Style {
									return ui.TypeBody.Merge(core.Style{FontWeight: core.FontWeightMedium, Color: colors.Text()})
								},
							},
							core.Text{
								ContentFunc: func() string {
									t := item()
									sub := formatTaskSubtitle(t)
									if chunk := formatChunkSummary(t); chunk != "" {
										return sub + " · " + chunk
									}
									return sub
								},
								StyleFunc: func() core.Style {
									return ui.TypeCaption.Merge(core.Style{Color: colors.Muted()})
								},
							},
						},
					},
					core.View{
						Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6},
						Children: []core.Node{
							ui.Badge{
								TextFunc: func() string {
									t := item()
									if t == nil {
										return ""
									}
									label, _ := statusMeta(t.Status)
									return label
								},
								VariantFunc: func() ui.BadgeVariant {
									t := item()
									if t == nil {
										return ui.BadgeNeutral
									}
									_, v := statusMeta(t.Status)
									return v
								},
							},
							core.Show{
								When: func() bool {
									t := item()
									return t != nil && (t.Status == task.StatusDownloading || t.Status == task.StatusQueued)
								},
								Then: func(cx *core.Scope) core.Node {
									return ui.Button{
										Icon:    lucide.Pause,
										Label:   "暂停",
										Variant: ui.ButtonGhost,
										Size:    ui.SizeSmall,
										OnPress: func() {
											if t := item(); t != nil {
												_ = state.backend.PauseTask(t.ID)
											}
										},
									}
								},
							},
							core.Show{
								When: func() bool {
									t := item()
									return t != nil && t.Status == task.StatusPaused
								},
								Then: func(cx *core.Scope) core.Node {
									return ui.Button{
										Icon:    lucide.Play,
										Label:   "继续",
										Variant: ui.ButtonGhost,
										Size:    ui.SizeSmall,
										OnPress: func() {
											if t := item(); t != nil {
												_ = state.backend.ResumeTask(t.ID)
											}
										},
									}
								},
							},
							core.Show{
								When: func() bool {
									t := item()
									return t != nil && t.Status == task.StatusError
								},
								Then: func(cx *core.Scope) core.Node {
									return ui.Button{
										Icon:    lucide.RotateCcw,
										Label:   "重试",
										Variant: ui.ButtonGhost,
										Size:    ui.SizeSmall,
										OnPress: func() {
											if t := item(); t != nil {
												_ = state.backend.RetryTask(t.ID)
											}
										},
									}
								},
							},
							core.Show{
								When: func() bool {
									t := item()
									return t != nil && t.Status == task.StatusCompleted
								},
								Then: func(cx *core.Scope) core.Node {
									return ui.Button{
										Icon:    lucide.ExternalLink,
										Label:   "打开文件",
										Variant: ui.ButtonGhost,
										Size:    ui.SizeSmall,
										OnPress: func() {
											if t := item(); t != nil {
												_ = state.backend.OpenFile(filepath.Join(t.Directory, t.Filename))
											}
										},
									}
								},
							},
							ui.Button{
								Icon:    lucide.Activity,
								Label:   "查看进度",
								Variant: ui.ButtonGhost,
								Size:    ui.SizeSmall,
								OnPress: func() {
									if t := item(); t != nil {
										state.showProgressWindow(t.ID)
									}
								},
							},
							ui.Button{
								Icon:    lucide.FolderOpen,
								Label:   "打开文件夹",
								Variant: ui.ButtonGhost,
								Size:    ui.SizeSmall,
								OnPress: func() {
									if t := item(); t != nil {
										_ = state.backend.OpenFolder(t.Directory)
									}
								},
							},
							ui.Button{
								Icon:    lucide.Trash,
								Label:   "删除任务",
								Variant: ui.ButtonGhost,
								Size:    ui.SizeSmall,
								OnPress: func() {
									if t := item(); t != nil {
										_ = state.backend.DeleteTask(t.ID, false)
									}
								},
							},
						},
					},
				},
			},
			core.Show{
				When: func() bool {
					t := item()
					return t != nil && t.Status != task.StatusCompleted
				},
				Then: func(cx *core.Scope) core.Node {
					return ui.Progress{
						Value: func() float32 { return taskProgressPercent(item()) },
						Indeterminate: func() bool {
							t := item()
							return t != nil && (t.Status == task.StatusProcessing || (t.Status == task.StatusDownloading && t.TotalBytes <= 0))
						},
					}
				},
			},
		},
	}
}
