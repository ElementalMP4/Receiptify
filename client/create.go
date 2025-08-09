package main

import (
	"encoding/json"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

var creatorComponents []Component
var creatorContainer *fyne.Container

func LoadTemplateIntoCreator(tmpl []Component) {
	creatorComponents = make([]Component, len(tmpl))
	copy(creatorComponents, tmpl)
	creatorContainer.Objects = nil

	for i, c := range creatorComponents {
		switch c.Type {
		case TextComponent:
			entry := widget.NewEntry()
			entry.SetText(c.Content)
			entry.MultiLine = true
			entry.Wrapping = fyne.TextWrapWord
			entry.TextStyle.Bold = c.Bold
			entry.TextStyle.Italic = c.Italic
			entry.Resize(fyne.NewSize(300, 30))
			idx := i
			entry.OnChanged = func(s string) {
				creatorComponents[idx].Content = s
			}
			creatorContainer.Add(container.NewVBox(
				widget.NewLabel(c.Name),
				entry,
			))
		case DividerComponent:
			line := canvas.NewRectangle(color.Black)
			line.SetMinSize(fyne.NewSize(300, float32(c.LineWidth)))
			creatorContainer.Add(line)
		}
	}
	creatorContainer.Refresh()
}

func CreateUI(w fyne.Window) fyne.CanvasObject {
	creatorContainer = container.NewVBox(widget.NewLabel("No template loaded."))

	loadBtn := widget.NewButton("Load Template", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			var imported []Component
			jsonParser := json.NewDecoder(reader)
			if err := jsonParser.Decode(&imported); err != nil {
				dialog.ShowError(err, w)
				return
			}

			LoadTemplateIntoCreator(imported)
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
		fd.Show()
	})

	exportBtn := widget.NewButton("Export JSON", func() {
		if len(creatorComponents) == 0 {
			dialog.ShowInformation("No Template", "Load a template first.", w)
			return
		}
		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			defer writer.Close()
			j, err := json.MarshalIndent(creatorComponents, "", "  ")
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			_, err = writer.Write(j)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation("Success", "JSON exported successfully.", w)
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
		fd.Show()
	})

	printBtn := widget.NewButton("Print", func() {
		if len(creatorComponents) == 0 {
			dialog.ShowInformation("No Template", "Load a template first.", w)
			return
		}
		err := SendToPrinter(creatorComponents, settings.PrintServerURL)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Printed", "Receipt has been printed!", w)
		}
	})

	buttons := container.NewHBox(loadBtn, exportBtn, printBtn)

	return container.NewVBox(
		MakeHeaderLabel("Receipt Creator"),
		buttons,
		widget.NewSeparator(),
		creatorContainer,
	)
}
