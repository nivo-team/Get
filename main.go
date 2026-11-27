package main

import (
	"context"
	_ "embed"

	"sheep-get/internal/clipboard"
	"sheep-get/internal/config"
	"sheep-get/internal/server"
	"sheep-get/internal/task"
	"sheep-get/internal/ui"
	"sheep-get/internal/window"
)

//go:embed build/appicon.png
var trayIconPNG []byte

type appUIBackend struct {
	app *App
}

func (b *appUIBackend) ListTasks() ([]*task.Task, error) {
	return b.app.ListTasks()
}

func (b *appUIBackend) PauseTask(id string) error {
	return b.app.PauseTask(id)
}

func (b *appUIBackend) ResumeTask(id string) error {
	return b.app.ResumeTask(id)
}

func (b *appUIBackend) RetryTask(id string) error {
	return b.app.RetryTask(id)
}

func (b *appUIBackend) DeleteTask(id string, deleteDiskFile bool) error {
	return b.app.DeleteTask(id, deleteDiskFile)
}

func (b *appUIBackend) OpenFile(path string) error {
	return b.app.OpenFile(path)
}

func (b *appUIBackend) OpenFolder(path string) error {
	return b.app.OpenFolder(path)
}

func (b *appUIBackend) OpenNewDownloadWithClipboard(text string) error {
	_, err := b.app.OpenNewDownloadWithClipboard(text)
	return err
}

func (b *appUIBackend) GetActiveFileInfo() (*window.FileInfoItem, error) {
	return b.app.GetActiveFileInfo()
}

func (b *appUIBackend) SubmitFileInfo(sub window.FileInfoSubmission) error {
	_, err := b.app.SubmitFileInfo(sub)
	return err
}

func (b *appUIBackend) CancelCurrentFileInfo() error {
	return b.app.CancelCurrentFileInfo()
}

func (b *appUIBackend) SwitchFileInfoActive(index int) (*window.FileInfoItem, error) {
	return b.app.SwitchFileInfoActive(index)
}

func (b *appUIBackend) SetFileInfoWindowView(view window.WindowView) {
	b.app.SetFileInfoWindowView(view)
}

func (b *appUIBackend) StartURL(rawURL, filename string) error {
	return b.app.StartDirectDownload(rawURL, filename, "")
}

func (b *appUIBackend) GetServerStatus() server.Status {
	return b.app.GetServerStatus()
}

func (b *appUIBackend) GetSettings() config.Settings {
	return b.app.GetSettings()
}

func (b *appUIBackend) UpdateSettings(s config.Settings) (config.Settings, error) {
	return b.app.UpdateSettings(s)
}

func (b *appUIBackend) OpenExtensionFolder() error {
	return b.app.OpenExtensionFolder()
}

func (b *appUIBackend) OpenSkillDirectory() error {
	return b.app.OpenSkillDirectory()
}

func (b *appUIBackend) AttachClipboard(reader clipboard.Reader) {
	b.app.AttachClipboard(reader)
}

func (b *appUIBackend) SetOnTaskChanged(fn func()) {
	b.app.SetOnTaskChanged(fn)
}

func (b *appUIBackend) SetOnShowProgress(fn func(taskID string)) {
	b.app.SetOnShowProgress(fn)
}

func main() {
	app := NewApp()
	app.StartServices(context.Background())
	defer app.Shutdown()

	ui.Run(&appUIBackend{app: app}, trayIconPNG)
}
