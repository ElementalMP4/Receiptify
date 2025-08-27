package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var settingsFile string
var settings AppSettings
var testPrint = []Component{
	{
		Type:     TextComponent,
		Content:  "Hello World!",
		Bold:     true,
		FontSize: "32",
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
		FontSize: "14",
	},
}

func getDefaultConfigPath() string {
	cwd, _ := os.Getwd()
	cwdSettings := filepath.Join(cwd, "settings.json")
	if _, err := os.Stat(cwdSettings); err == nil {
		return cwdSettings
	}

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Receiptify", "settings.json")
	}
	return filepath.Join(home, ".config", "receiptify", "settings.json")
}

func ensureConfigFileExists(path string) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		baseDir := strings.TrimSuffix(path, string(filepath.Separator)+"settings.json")
		pluginDir := filepath.Join(baseDir, "plugins")

		emptySettings := AppSettings{
			PluginPath: pluginDir,
		}

		data, err := json.MarshalIndent(emptySettings, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, data, 0644)
	}

	return nil
}

func LoadSettings() {
	settingsFile = getDefaultConfigPath()
	_ = ensureConfigFileExists(settingsFile)
	data, err := os.ReadFile(settingsFile)
	if err == nil {
		json.Unmarshal(data, &settings)
	}
}

func SaveSettings(reloadLua bool, w fyne.Window) {
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsFile, data, 0644)

	if reloadLua {
		err := ConfigureLuaAndLoadPlugins()
		if err != nil && err.Error() != "plugin path not set" {
			dialog.ShowError(err, w)
		}
	}
}

func SettingsUI(w fyne.Window) fyne.CanvasObject {
	urlEntry := widget.NewEntry()
	urlEntry.SetText(settings.PrintServerURL)

	pluginPathEntry := widget.NewEntry()
	pluginPathEntry.SetText(settings.PluginPath)

	saveBtn := widget.NewButton("Save", func() {
		settings.PrintServerURL = urlEntry.Text
		settings.PluginPath = pluginPathEntry.Text
		SaveSettings(true, w)
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
			widget.NewFormItem("Plugin Path", pluginPathEntry),
			widget.NewFormItem("Test", testPrinterBtn),
		),
		saveBtn,
	)
}
