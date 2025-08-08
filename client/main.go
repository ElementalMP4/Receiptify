package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.NewWithID("Receiptify")
	w := a.NewWindow("Receiptify")
	w.Resize(fyne.NewSize(900, 700))

	loadSettings()

	w.SetContent(mainAppContent(w))
	w.ShowAndRun()
}

// Helper to return the main app content (your navigation etc.)
func mainAppContent(w fyne.Window) fyne.CanvasObject {
	content := container.NewStack()
	var btnEditor, btnSettings *widget.Button
	var navButtons *fyne.Container

	setActive := func(active string) {
		btnEditor.Importance = widget.MediumImportance
		btnSettings.Importance = widget.MediumImportance

		switch active {
		case "editor":
			btnEditor.Importance = widget.HighImportance
			content.Objects = []fyne.CanvasObject{EditorUI(w)}
		case "settings":
			btnSettings.Importance = widget.HighImportance
			content.Objects = []fyne.CanvasObject{SettingsUI(w)}
		}
		content.Refresh()
		navButtons.Refresh()
	}

	btnEditor = widget.NewButtonWithIcon("Editor", theme.DocumentCreateIcon(), func() { setActive("editor") })
	btnSettings = widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() { setActive("settings") })

	navButtons = container.NewVBox(
		MakeHeaderLabel("Receiptify"),
		btnEditor,
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
