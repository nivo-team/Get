package ui

import (
	"strconv"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"sheep-get/internal/config"
)

type settingsState struct {
	theme                         *core.Signal[config.ThemeMode]
	defaultDir                    *core.Signal[string]
	tempDir                       *core.Signal[string]
	maxConcurrent                 *core.Signal[string]
	defaultConnections            *core.Signal[string]
	speedLimitKB                  *core.Signal[string]
	duplicatePolicy               *core.Signal[config.DuplicateURLPolicy]
	preDownload                   *core.Signal[bool]
	skipFileInfoConfirm           *core.Signal[bool]
	showProgressWindow            *core.Signal[bool]
	keepCompletedInfo             *core.Signal[bool]
	focusProgressWindowOnComplete *core.Signal[bool]

	launchAtStartup      *core.Signal[bool]
	silentStartup        *core.Signal[bool]
	lightweightMode      *core.Signal[bool]
	checkUpdateOnStartup *core.Signal[bool]
	clipboardEnabled     *core.Signal[bool]
	enableLogging        *core.Signal[bool]

	proxyMode  *core.Signal[config.ProxyMode]
	proxyAddrs *core.Signal[string]

	serverPort *core.Signal[string]
	apiKey     *core.Signal[string]

	saving   *core.Signal[bool]
	feedback *core.Signal[string]
	isError  *core.Signal[bool]
	snapshot config.Settings
}

func newSettingsState(app *core.Scope, s config.Settings) *settingsState {
	st := &settingsState{
		theme:                         core.NewSignal(app, s.Appearance.Theme),
		defaultDir:                    core.NewSignal(app, s.Download.DefaultDirectory),
		tempDir:                       core.NewSignal(app, s.Download.TempDirectory),
		maxConcurrent:                 core.NewSignal(app, strconv.Itoa(s.Download.MaxConcurrentDownloads)),
		defaultConnections:            core.NewSignal(app, strconv.Itoa(s.Download.DefaultConnectionsPerTask)),
		speedLimitKB:                  core.NewSignal(app, strconv.Itoa(s.Download.SpeedLimitKB)),
		duplicatePolicy:               core.NewSignal(app, s.Download.DuplicateURLPolicy),
		preDownload:                   core.NewSignal(app, s.Download.PreDownload),
		skipFileInfoConfirm:           core.NewSignal(app, s.Download.SkipFileInfoConfirm),
		showProgressWindow:            core.NewSignal(app, s.Download.ShowProgressWindow),
		keepCompletedInfo:             core.NewSignal(app, s.Download.KeepCompletedInfo),
		focusProgressWindowOnComplete: core.NewSignal(app, s.Download.FocusProgressWindowOnComplete),
		launchAtStartup:               core.NewSignal(app, s.General.LaunchAtStartup),
		silentStartup:                 core.NewSignal(app, s.General.SilentStartup),
		lightweightMode:               core.NewSignal(app, s.General.LightweightMode),
		checkUpdateOnStartup:          core.NewSignal(app, s.General.CheckUpdateOnStartup),
		clipboardEnabled:              core.NewSignal(app, s.Clipboard.Enabled),
		enableLogging:                 core.NewSignal(app, s.General.EnableLogging),
		proxyMode:                     core.NewSignal(app, s.Proxy.Mode),
		proxyAddrs:                    core.NewSignal(app, strings.Join(s.Proxy.EffectiveCustomAddrs(), ", ")),
		serverPort:                    core.NewSignal(app, strconv.Itoa(s.General.ServerPort)),
		apiKey:                        core.NewSignal(app, s.General.APIKey),
		saving:                        core.NewSignal(app, false),
		feedback:                      core.NewSignal(app, ""),
		isError:                       core.NewSignal(app, false),
		snapshot:                      s,
	}
	return st
}

func (st *settingsState) load(s config.Settings) {
	st.snapshot = s
	st.theme.Set(s.Appearance.Theme)
	st.defaultDir.Set(s.Download.DefaultDirectory)
	st.tempDir.Set(s.Download.TempDirectory)
	st.maxConcurrent.Set(strconv.Itoa(s.Download.MaxConcurrentDownloads))
	st.defaultConnections.Set(strconv.Itoa(s.Download.DefaultConnectionsPerTask))
	st.speedLimitKB.Set(strconv.Itoa(s.Download.SpeedLimitKB))
	st.duplicatePolicy.Set(s.Download.DuplicateURLPolicy)
	st.preDownload.Set(s.Download.PreDownload)
	st.skipFileInfoConfirm.Set(s.Download.SkipFileInfoConfirm)
	st.showProgressWindow.Set(s.Download.ShowProgressWindow)
	st.keepCompletedInfo.Set(s.Download.KeepCompletedInfo)
	st.focusProgressWindowOnComplete.Set(s.Download.FocusProgressWindowOnComplete)
	st.launchAtStartup.Set(s.General.LaunchAtStartup)
	st.silentStartup.Set(s.General.SilentStartup)
	st.lightweightMode.Set(s.General.LightweightMode)
	st.checkUpdateOnStartup.Set(s.General.CheckUpdateOnStartup)
	st.clipboardEnabled.Set(s.Clipboard.Enabled)
	st.enableLogging.Set(s.General.EnableLogging)
	st.proxyMode.Set(s.Proxy.Mode)
	st.proxyAddrs.Set(strings.Join(s.Proxy.EffectiveCustomAddrs(), ", "))
	st.serverPort.Set(strconv.Itoa(s.General.ServerPort))
	st.apiKey.Set(s.General.APIKey)
}

func (st *settingsState) buildConfig() (config.Settings, error) {
	s := st.snapshot
	s.Appearance.Theme = st.theme.Get()

	s.Download.DefaultDirectory = strings.TrimSpace(st.defaultDir.Get())
	s.Download.TempDirectory = strings.TrimSpace(st.tempDir.Get())

	if v, err := parsePositiveInt(st.maxConcurrent.Get(), "最大同时下载数"); err != nil {
		return s, err
	} else {
		s.Download.MaxConcurrentDownloads = v
	}
	if v, err := parsePositiveInt(st.defaultConnections.Get(), "单任务并发连接数"); err != nil {
		return s, err
	} else {
		s.Download.DefaultConnectionsPerTask = v
	}
	if v, err := parseNonNegativeInt(st.speedLimitKB.Get(), "全局限速"); err != nil {
		return s, err
	} else {
		s.Download.SpeedLimitKB = v
	}

	s.Download.DuplicateURLPolicy = st.duplicatePolicy.Get()
	s.Download.PreDownload = st.preDownload.Get()
	s.Download.SkipFileInfoConfirm = st.skipFileInfoConfirm.Get()
	s.Download.ShowProgressWindow = st.showProgressWindow.Get()
	s.Download.KeepCompletedInfo = st.keepCompletedInfo.Get()
	s.Download.FocusProgressWindowOnComplete = st.focusProgressWindowOnComplete.Get()

	s.General.LaunchAtStartup = st.launchAtStartup.Get()
	s.General.SilentStartup = st.silentStartup.Get()
	s.General.LightweightMode = st.lightweightMode.Get()
	s.General.CheckUpdateOnStartup = st.checkUpdateOnStartup.Get()
	s.General.EnableLogging = st.enableLogging.Get()
	s.Clipboard.Enabled = st.clipboardEnabled.Get()

	if port, err := parsePositiveInt(st.serverPort.Get(), "本地服务端口"); err != nil {
		return s, err
	} else {
		s.General.ServerPort = port
	}
	s.General.APIKey = strings.TrimSpace(st.apiKey.Get())

	s.Proxy.Mode = st.proxyMode.Get()
	var addrs []string
	for _, part := range strings.Split(st.proxyAddrs.Get(), ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			addrs = append(addrs, trimmed)
		}
	}
	s.Proxy.CustomAddrs = addrs
	if len(addrs) > 0 {
		s.Proxy.CustomAddr = addrs[0]
	} else {
		s.Proxy.CustomAddr = ""
	}
	return s, nil
}

func parsePositiveInt(raw, label string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return 0, &settingsValidationError{msg: label + "必须为正整数"}
	}
	return v, nil
}

func parseNonNegativeInt(raw, label string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v < 0 {
		return 0, &settingsValidationError{msg: label + "不能为负数"}
	}
	return v, nil
}

type settingsValidationError struct{ msg string }

func (e *settingsValidationError) Error() string { return e.msg }

func (s *appState) saveSettings(cx *core.Scope) {
	st := s.settings
	if st.saving.Get() {
		return
	}
	cfg, err := st.buildConfig()
	if err != nil {
		st.isError.Set(true)
		st.feedback.Set(err.Error())
		return
	}
	st.saving.Set(true)
	st.isError.Set(false)
	st.feedback.Set("")

	go func() {
		updated, err := s.backend.UpdateSettings(cfg)
		if s.app == nil {
			return
		}
		s.app.Post(func() {
			st.saving.Set(false)
			if err != nil {
				st.isError.Set(true)
				st.feedback.Set("保存失败：" + err.Error())
				return
			}
			st.load(updated)
			st.isError.Set(false)
			st.feedback.Set("设置已保存并生效")
			applyThemeMode(cx, updated.Appearance.Theme)
			s.refresh()
		})
	}()
}

func applyThemeMode(cx *core.Scope, mode config.ThemeMode) {
	if cx == nil {
		return
	}
	switch mode {
	case config.ThemeLight:
		cx.SetAppearance(core.AppearanceLight)
	case config.ThemeDark:
		cx.SetAppearance(core.AppearanceDark)
	default:
		cx.SetAppearance(core.AppearanceSystem)
	}
}

// SettingsView renders the application settings page inside the main window.
func SettingsView(cx *core.Scope, state *appState) core.Node {
	colors := ui.ColorsOf(cx)
	st := state.settings

	return core.View{
		Style: core.Style{
			FlexGrow:   1,
			FlexShrink: 1,
			MinHeight:  core.Pt(0),
			Overflow:   core.Hidden,
		},
		Children: []core.Node{
			// Top action bar for saving & feedback
			core.View{
				StyleFunc: func() core.Style {
					return core.Style{
						Direction:      core.Row,
						AlignItems:     core.AlignCenter,
						JustifyContent: core.JustifySpaceBetween,
						Padding:        core.Symmetric(10, 16),
						Borders:        core.Edges{Bottom: 1},
						BorderColor:    colors.Line(),
					}
				},
				Children: []core.Node{
					core.Text{
						ContentFunc: func() string {
							if msg := st.feedback.Get(); msg != "" {
								return msg
							}
							return "修改后点击「保存设置」应用到运行中的引擎与服务"
						},
						StyleFunc: func() core.Style {
							c := colors.Muted()
							if st.isError.Get() {
								c = colors.Danger()
							} else if st.feedback.Get() != "" {
								c = colors.Up()
							}
							return ui.TypeCaption.Merge(core.Style{Color: c})
						},
					},
					core.View{
						Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
						Children: []core.Node{
							ui.Button{
								Text:    "重置",
								Variant: ui.ButtonGhost,
								Size:    ui.SizeSmall,
								Style:   core.Style{Height: core.Pt(headerButtonHeight)},
								OnPress: func() {
									st.load(state.backend.GetSettings())
									st.isError.Set(false)
									st.feedback.Set("已还原为当前生效设置")
								},
							},
							ui.Button{
								Text:    "保存设置",
								Icon:    lucide.Save,
								Variant: ui.ButtonPrimary,
								Size:    ui.SizeSmall,
								Style:   core.Style{Height: core.Pt(headerButtonHeight)},
								Busy:    st.saving.Get,
								OnPress: func() { state.saveSettings(cx) },
							},
						},
					},
				},
			},
			ui.ScrollView{
				Style:        core.Style{FlexGrow: 1, FlexShrink: 1, MinHeight: core.Pt(0)},
				ContentStyle: core.Style{Padding: core.All(16), Gap: 16},
				Children: []core.Node{
					// 1. 外观与主题
					ui.FormSection{
						Title: "外观与主题",
						Children: []core.Node{
							ui.FormRow{
								Title:  "界面主题",
								Detail: "立即预览浅色、深色或跟随系统外观",
								Control: core.View{
									Style: core.Style{Direction: core.Row, Gap: 6},
									Children: []core.Node{
										themeOptionButton(cx, st, config.ThemeSystem, "跟随系统"),
										themeOptionButton(cx, st, config.ThemeLight, "浅色"),
										themeOptionButton(cx, st, config.ThemeDark, "深色"),
									},
								},
							},
						},
					},

					// 2. 下载与行为
					ui.FormSection{
						Title: "下载与行为",
						Children: []core.Node{
							ui.FormRow{
								Title: "默认下载目录",
								Control: core.View{
									Style:    core.Style{Width: core.Pt(260)},
									Children: []core.Node{ui.Input{Value: st.defaultDir.Get, OnChange: st.defaultDir.Set}},
								},
							},
							ui.FormRow{
								Title: "临时分片目录",
								Control: core.View{
									Style:    core.Style{Width: core.Pt(260)},
									Children: []core.Node{ui.Input{Value: st.tempDir.Get, OnChange: st.tempDir.Set}},
								},
							},
							ui.FormRow{
								Title:  "最大同时下载数",
								Detail: "同时进行传输的任务数量上限",
								Control: core.View{
									Style:    core.Style{Width: core.Pt(90)},
									Children: []core.Node{ui.Input{Value: st.maxConcurrent.Get, OnChange: st.maxConcurrent.Set}},
								},
							},
							ui.FormRow{
								Title:  "单任务默认并发数",
								Detail: "HTTP 多线程分块默认连接数",
								Control: core.View{
									Style:    core.Style{Width: core.Pt(90)},
									Children: []core.Node{ui.Input{Value: st.defaultConnections.Get, OnChange: st.defaultConnections.Set}},
								},
							},
							ui.FormRow{
								Title:  "全局限速 (KB/s)",
								Detail: "设为 0 表示不限制下载速度",
								Control: core.View{
									Style:    core.Style{Width: core.Pt(90)},
									Children: []core.Node{ui.Input{Value: st.speedLimitKB.Get, OnChange: st.speedLimitKB.Set}},
								},
							},
							ui.FormRow{
								Title:  "重复链接策略",
								Detail: "当添加已存在下载记录的链接时",
								Control: core.View{
									Style: core.Style{Direction: core.Row, Gap: 4},
									Children: []core.Node{
										dupPolicyButton(st, config.DuplicatePolicyPrompt, "询问"),
										dupPolicyButton(st, config.DuplicatePolicySkipShowCompleted, "跳过"),
										dupPolicyButton(st, config.DuplicatePolicyContinueOverwrite, "续传/覆盖"),
										dupPolicyButton(st, config.DuplicatePolicyNumberedCopy, "序号副本"),
									},
								},
							},
							ui.FormRow{
								Title:   "提前下载",
								Detail:  "打开文件信息确认窗口时在后台提前开始传输",
								Control: ui.Switch{Checked: st.preDownload.Get, OnChange: st.preDownload.Set, Label: "提前下载"},
							},
							ui.FormRow{
								Title:   "跳过文件信息确认窗口",
								Detail:  "接管下载时直接入队，不再弹出新建下载确认框",
								Control: ui.Switch{Checked: st.skipFileInfoConfirm.Get, OnChange: st.skipFileInfoConfirm.Set, Label: "跳过文件信息确认窗口"},
							},
							ui.FormRow{
								Title:   "自动显示下载进度窗口",
								Control: ui.Switch{Checked: st.showProgressWindow.Get, OnChange: st.showProgressWindow.Set, Label: "自动显示下载进度窗口"},
							},
							ui.FormRow{
								Title:   "保留已完成信息",
								Control: ui.Switch{Checked: st.keepCompletedInfo.Get, OnChange: st.keepCompletedInfo.Set, Label: "保留已完成信息"},
							},
							ui.FormRow{
								Title:   "完成后聚焦进度窗口",
								Control: ui.Switch{Checked: st.focusProgressWindowOnComplete.Get, OnChange: st.focusProgressWindowOnComplete.Set, Label: "完成后聚焦进度窗口"},
							},
						},
					},

					// 3. 常规与系统
					ui.FormSection{
						Title: "常规与系统",
						Children: []core.Node{
							ui.FormRow{
								Title:   "开机自启动",
								Control: ui.Switch{Checked: st.launchAtStartup.Get, OnChange: st.launchAtStartup.Set, Label: "开机自启动"},
							},
							ui.FormRow{
								Title:   "静默启动",
								Detail:  "启动后最小化驻留托盘，不自动弹出主窗口",
								Control: ui.Switch{Checked: st.silentStartup.Get, OnChange: st.silentStartup.Set, Label: "静默启动"},
							},
							ui.FormRow{
								Title:   "轻量模式",
								Detail:  "静默启动或关闭窗口后尽量释放渲染资源",
								Control: ui.Switch{Checked: st.lightweightMode.Get, OnChange: st.lightweightMode.Set, Label: "轻量模式"},
							},
							ui.FormRow{
								Title:   "启动时检查更新",
								Control: ui.Switch{Checked: st.checkUpdateOnStartup.Get, OnChange: st.checkUpdateOnStartup.Set, Label: "启动时检查更新"},
							},
							ui.FormRow{
								Title:   "剪贴板链接监听",
								Detail:  "自动识别复制的 HTTP/HTTPS 下载链接",
								Control: ui.Switch{Checked: st.clipboardEnabled.Get, OnChange: st.clipboardEnabled.Set, Label: "剪贴板链接监听"},
							},
							ui.FormRow{
								Title:   "启用运行日志",
								Detail:  "排查问题时将运行日志写入本地 logs 目录",
								Control: ui.Switch{Checked: st.enableLogging.Get, OnChange: st.enableLogging.Set, Label: "启用运行日志"},
							},
						},
					},

					// 4. 网络代理
					ui.FormSection{
						Title: "网络代理",
						Children: []core.Node{
							ui.FormRow{
								Title: "代理模式",
								Control: core.View{
									Style: core.Style{Direction: core.Row, Gap: 6},
									Children: []core.Node{
										proxyModeButton(st, config.ProxyModeDirect, "直连"),
										proxyModeButton(st, config.ProxyModeSystem, "系统代理"),
										proxyModeButton(st, config.ProxyModeCustom, "自定义"),
									},
								},
							},
							core.Show{
								When: func() bool { return st.proxyMode.Get() == config.ProxyModeCustom },
								Then: func(cx *core.Scope) core.Node {
									return ui.FormRow{
										Title:  "自定义代理地址",
										Detail: "支持 http / https / socks5，多个地址用逗号分隔",
										Control: core.View{
											Style: core.Style{Width: core.Pt(260)},
											Children: []core.Node{
												ui.Input{
													Value:       st.proxyAddrs.Get,
													OnChange:    st.proxyAddrs.Set,
													Placeholder: "http://127.0.0.1:7890",
												},
											},
										},
									}
								},
							},
						},
					},

					// 5. 本地服务与集成
					ui.FormSection{
						Title: "本地服务与生态集成",
						Children: []core.Node{
							ui.FormRow{
								Title:  "本地服务端口",
								Detail: "浏览器扩展与开放 API 通信端口（默认 9248）",
								Control: core.View{
									Style:    core.Style{Width: core.Pt(90)},
									Children: []core.Node{ui.Input{Value: st.serverPort.Get, OnChange: st.serverPort.Set}},
								},
							},
							ui.FormRow{
								Title:  "静态 API Key",
								Detail: "供外部脚本或 AI Agent 调用本地接口鉴权使用",
								Control: core.View{
									Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6},
									Children: []core.Node{
										core.View{
											Style:    core.Style{Width: core.Pt(200)},
											Children: []core.Node{ui.Input{Value: st.apiKey.Get, OnChange: st.apiKey.Set, Placeholder: "未设置"}},
										},
										ui.Button{
											Text:    "生成",
											Icon:    lucide.Key,
											Variant: ui.ButtonSecondary,
											Size:    ui.SizeSmall,
											OnPress: func() { st.apiKey.Set(config.GenerateRandomToken()) },
										},
									},
								},
							},
							ui.FormRow{
								Title:  "配套浏览器扩展",
								Detail: "在文件管理器中定位扩展目录以拖入 Chrome / Edge",
								Control: ui.Button{
									Text:    "定位扩展目录",
									Icon:    lucide.Folder,
									Variant: ui.ButtonSecondary,
									Size:    ui.SizeSmall,
									OnPress: func() {
										if err := state.backend.OpenExtensionFolder(); err != nil {
											st.isError.Set(true)
											st.feedback.Set("定位扩展目录失败：" + err.Error())
										}
									},
								},
							},
							ui.FormRow{
								Title:  "AI Agent Skill",
								Detail: "定位内置 Agent Skill 目录以便导入 AI 编程工具",
								Control: ui.Button{
									Text:    "定位 Skill 目录",
									Icon:    lucide.Folder,
									Variant: ui.ButtonSecondary,
									Size:    ui.SizeSmall,
									OnPress: func() {
										if err := state.backend.OpenSkillDirectory(); err != nil {
											st.isError.Set(true)
											st.feedback.Set("定位 Skill 目录失败：" + err.Error())
										}
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func themeOptionButton(cx *core.Scope, st *settingsState, mode config.ThemeMode, label string) core.Node {
	return core.Dynamic(func(*core.Scope) core.Node {
		v := ui.ButtonSecondary
		if st.theme.Get() == mode {
			v = ui.ButtonPrimary
		}
		return ui.Button{
			Text:    label,
			Variant: v,
			Size:    ui.SizeSmall,
			OnPress: func() {
				st.theme.Set(mode)
				applyThemeMode(cx, mode)
			},
		}
	})
}

func dupPolicyButton(st *settingsState, policy config.DuplicateURLPolicy, label string) core.Node {
	return core.Dynamic(func(*core.Scope) core.Node {
		v := ui.ButtonSecondary
		if st.duplicatePolicy.Get() == policy {
			v = ui.ButtonPrimary
		}
		return ui.Button{
			Text:    label,
			Variant: v,
			Size:    ui.SizeSmall,
			OnPress: func() { st.duplicatePolicy.Set(policy) },
		}
	})
}

func proxyModeButton(st *settingsState, mode config.ProxyMode, label string) core.Node {
	return core.Dynamic(func(*core.Scope) core.Node {
		v := ui.ButtonSecondary
		if st.proxyMode.Get() == mode {
			v = ui.ButtonPrimary
		}
		return ui.Button{
			Text:    label,
			Variant: v,
			Size:    ui.SizeSmall,
			OnPress: func() { st.proxyMode.Set(mode) },
		}
	})
}
