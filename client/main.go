package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var setActive func(string)

func main() {
	a := app.NewWithID("Receiptify")
	w := a.NewWindow("Receiptify")
	w.Resize(fyne.NewSize(900, 700))

	LoadSettings()

	err := ConfigureLuaAndLoadPlugins()
	if err != nil && err.Error() != "plugin path not set" {
		log.Fatalf("An error occurred whilst loading plugins: %v", err)
	}

	w.SetContent(mainAppContent(w))
	w.ShowAndRun()

	if luaVm != nil && !luaVm.Dead {
		luaVm.Close()
	}
}

func mainAppContent(w fyne.Window) fyne.CanvasObject {
	content := container.NewStack()
	var btnEditor, btnSettings, btnLibrary, btnCreate *widget.Button
	var navButtons *fyne.Container

	setActive = func(active string) {
		btnEditor.Importance = widget.MediumImportance
		btnSettings.Importance = widget.MediumImportance
		btnLibrary.Importance = widget.MediumImportance
		btnCreate.Importance = widget.MediumImportance

		switch active {
		case "editor":
			btnEditor.Importance = widget.HighImportance
			content.Objects = []fyne.CanvasObject{EditorUI(w)}
		case "settings":
			btnSettings.Importance = widget.HighImportance
			content.Objects = []fyne.CanvasObject{SettingsUI(w)}
		case "library":
			btnLibrary.Importance = widget.HighImportance
			content.Objects = []fyne.CanvasObject{LibraryUI(w)}
		case "create":
			btnCreate.Importance = widget.HighImportance
			content.Objects = []fyne.CanvasObject{CreateUI(w)}
		}
		content.Refresh()
		navButtons.Refresh()
	}

	btnEditor = widget.NewButtonWithIcon("Template Builder", theme.DocumentCreateIcon(), func() { setActive("editor") })
	btnSettings = widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() { setActive("settings") })
	btnLibrary = widget.NewButtonWithIcon("Template Library", theme.FolderOpenIcon(), func() { setActive("library") })
	btnCreate = widget.NewButtonWithIcon("Create Receipt", theme.ContentAddIcon(), func() { setActive("create") })

	navButtons = container.NewVBox(
		MakeHeaderLabel("Receiptify"),
		btnEditor,
		btnCreate,
		btnLibrary,
		btnSettings,
		layout.NewSpacer(),
	)

	navContainer := container.NewBorder(nil, nil, nil, widget.NewSeparator(),
		container.NewStack(navButtons))

	split := container.NewHSplit(navContainer, content)
	split.Offset = 0.2

	setActive("editor")

	return split
}
