package main

import (
	"encoding/json"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var settingsFile = "settings.json"
var settings AppSettings
var testPrint = []Component{
	{
		Type:     TextComponent,
		Content:  "Hello World!",
		Bold:     true,
		FontSize: 32,
		Align:    "center",
	},
	{
		Type:      DividerComponent,
		LineWidth: 5,
	},
	{
		Type:     TextComponent,
		Content:  "Does this thing even work?",
		Italic:   true,
		FontSize: 14,
	},
}

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

	testPrinterBtn := widget.NewButton("Send Test Print", func() {
		urlToTest := urlEntry.Text
		err := SendToPrinter(testPrint, urlToTest)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Printed", "Test print was successful!", w)
		}
	})

	return container.NewVBox(
		MakeHeaderLabel("Settings"),
		widget.NewForm(
			widget.NewFormItem("Print Server URL", urlEntry),
			widget.NewFormItem("Test", testPrinterBtn),
		),
		saveBtn,
	)
}
