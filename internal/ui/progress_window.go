package ui

import (
	"fmt"
	"path/filepath"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"sheep-get/internal/task"
)

func openProgressWindow(app *core.Scope, state *appState) *desktop.Window {
	win := desktop.OpenWindow(app, desktop.WindowOptions{
		Title:  "下载进度 - SheepGet",
		Key:    "progress",
		Width:  560,
		Height: 360,
		Constraints: desktop.WindowConstraints{
			MinWidth:  460,
			MinHeight: 220,
			MaxHeight: 680,
		},
	}, func(cx *core.Scope) core.Node {
		return ProgressWindowView(cx, state)
	})
	if win != nil {
		win.Show()
		win.Focus()
	}
	return win
}

// ProgressWindowView renders the shared download progress window:
// completed items pinned on top, active downloads sorted by progress descending.
func ProgressWindowView(cx *core.Scope, state *appState) core.Node {
	if desktop.IsAvailable(cx) {
		desktop.SetTitleBar(cx, desktop.TitleBarHidden, ui.HeaderHeight)
	}
	hideSystemWindowControls(cx)
	colors := ui.ColorsOf(cx)
	return insetWindowShell(cx,
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
							Content: "实时下载进度",
							Style:   core.Style{FontSize: 13, FontWeight: core.FontWeightSemibold},
						},
						ui.Badge{
							TextFunc: func() string {
								comp, act := state.progressTasks()
								return fmt.Sprintf("%d 项", len(comp)+len(act))
							},
							Variant: ui.BadgeNeutral,
						},
					},
				},
				core.View{
					Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOff},
					Style:   core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6},
					Children: []core.Node{
						ui.Button{
							Text:    "清空已完成",
							Variant: ui.ButtonGhost,
							Size:    ui.SizeSmall,
							Style:   core.Style{Height: core.Pt(headerButtonHeight)},
							OnPress: state.dismissAllCompleted,
						},
						windowControlButtons(cx, false),
					},
				},
			},
		},
		core.Show{
			When: func() bool {
				comp, act := state.progressTasks()
				return len(comp)+len(act) == 0
			},
			Then: func(cx *core.Scope) core.Node {
				return core.View{
					Style: core.Style{
						FlexGrow:       1,
						AlignItems:     core.AlignCenter,
						JustifyContent: core.JustifyCenter,
						Gap:            8,
						Padding:        core.All(24),
					},
					Children: []core.Node{
						core.Icon{
							SVG: lucide.Activity(),
							StyleFunc: func() core.Style {
								return core.Style{Width: core.Pt(28), Height: core.Pt(28), Color: colors.Faint()}
							},
						},
						core.Text{
							Content: "当前没有进行中的下载任务",
							StyleFunc: func() core.Style {
								return ui.TypeCaption.Merge(core.Style{Color: colors.Muted()})
							},
						},
					},
				}
			},
			Else: func(cx *core.Scope) core.Node {
				return ui.ScrollView{
					Style:        core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: core.Pt(0)},
					ContentStyle: core.Style{Padding: core.All(12), Gap: 8},
					Children: []core.Node{
						core.Each(
							func() []*task.Task {
								comp, act := state.progressTasks()
								all := make([]*task.Task, 0, len(comp)+len(act))
								all = append(all, comp...)
								all = append(all, act...)
								return all
							},
							func(t *task.Task) string { return t.ID },
							func(cx *core.Scope, item func() *task.Task, _ func() int) core.Node {
								return progressItemCard(cx, state, item)
							},
						),
					},
				}
			},
		},
	)
}

func progressItemCard(cx *core.Scope, state *appState, item func() *task.Task) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{
		StyleFunc: func() core.Style {
			return core.Style{
				Padding:      core.Symmetric(10, 12),
				Gap:          6,
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
					Gap:            8,
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
						Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4},
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
									return core.View{
										Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4},
										Children: []core.Node{
											ui.Button{
												Icon:    lucide.ExternalLink,
												Label:   "打开文件",
												Variant: ui.ButtonGhost,
												Size:    ui.SizeSmall,
												OnPress: func() {
													if t := item(); t != nil {
														_ = state.backend.OpenFile(filepath.Join(t.Directory, t.Filename))
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
												Icon:    lucide.X,
												Label:   "关闭提示",
												Variant: ui.ButtonGhost,
												Size:    ui.SizeSmall,
												OnPress: func() {
													if t := item(); t != nil {
														state.dismissTask(t.ID)
													}
												},
											},
										},
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
