package ui

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

const headerButtonHeight core.Points = 24

// insetWindowShell wraps secondary windows (FileInfo, Progress) in the same
// outer chrome + 5pt inset rounded card as the main SidebarShellInset window.
func insetWindowShell(cx *core.Scope, children ...core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{
		StyleFunc: func() core.Style {
			return core.Style{
				FlexGrow:   1,
				FlexShrink: 1,
				MinHeight:  core.Pt(0),
				Padding:    core.All(5),
				Background: ui.Chrome(cx),
				Color:      colors.Text(),
			}
		},
		Children: []core.Node{
			core.View{
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
				Children: children,
			},
		},
	}
}

func hideSystemWindowControls(cx *core.Scope) {
	if !desktop.IsAvailable(cx) {
		return
	}
	cx.Effect(func() {
		desktop.SetWindowControls(cx, desktop.WindowControls{Hidden: true})
	})
	cx.Cleanup(func() {
		desktop.SetWindowControls(cx, desktop.WindowControls{})
	})
}

func windowControlButtons(cx *core.Scope, allowMaximize bool) core.Node {
	colors := ui.ColorsOf(cx)
	buttons := []core.Node{
		ui.Button{
			Icon:     lucide.Minus,
			Label:    "最小化",
			Variant:  ui.ButtonGhost,
			Size:     ui.SizeSmall,
			Side:     headerButtonHeight,
			IconSize: 14,
			OnPress:  func() { desktop.MinimizeWindow(cx) },
		},
	}
	if allowMaximize {
		buttons = append(buttons, ui.Button{
			Icon:     lucide.Square,
			Label:    "最大化或还原",
			Variant:  ui.ButtonGhost,
			Size:     ui.SizeSmall,
			Side:     headerButtonHeight,
			IconSize: 13,
			OnPress:  func() { desktop.ToggleMaximize(cx) },
		})
	}
	buttons = append(buttons, ui.Button{
		Icon:     lucide.X,
		Label:    "关闭窗口",
		Variant:  ui.ButtonGhost,
		Size:     ui.SizeSmall,
		Side:     headerButtonHeight,
		IconSize: 14,
		OnPress:  func() { desktop.CloseWindow(cx) },
	})

	return core.View{
		Desktop: core.DesktopViewProps{WindowDrag: core.WindowDragOff},
		Style: core.Style{
			Direction:  core.Row,
			AlignItems: core.AlignCenter,
			Gap:        4,
		},
		Children: []core.Node{
			core.View{
				StyleFunc: func() core.Style {
					return core.Style{
						Width:      core.Pt(1),
						Height:     core.Pt(14),
						Margin:     core.Symmetric(0, 4),
						Background: colors.Line(),
					}
				},
			},
			core.View{
				Style: core.Style{
					Direction:  core.Row,
					AlignItems: core.AlignCenter,
					Gap:        2,
				},
				Children: buttons,
			},
		},
	}
}
