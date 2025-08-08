package main

import (
	"encoding/json"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type AppSettings struct {
	PrintServerURL string `json:"print_server_url"`
}

var settingsFile = "settings.json"
var settings AppSettings

func loadSettings() {
	data, err := os.ReadFile(settingsFile)
	if err == nil {
		json.Unmarshal(data, &settings)
	}
}

func saveSettings() {
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsFile, data, 0644)
}

func SettingsUI(w fyne.Window) fyne.CanvasObject {
	urlEntry := widget.NewEntry()
	urlEntry.SetText(settings.PrintServerURL)

	saveBtn := widget.NewButton("Save", func() {
		settings.PrintServerURL = urlEntry.Text
		saveSettings()
		dialog.ShowInformation("Saved", "Settings saved!", w)
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(widget.NewFormItem("Print Server URL",
			urlEntry)),
		saveBtn,
	)
}
