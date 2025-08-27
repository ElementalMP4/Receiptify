package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/color"
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

var creatorComponents []Component
var creatorContainer *fyne.Container
var currentCreatorTemplate string

func isPluginCall(token string) bool {
	return strings.HasPrefix(token, "{{") && strings.HasSuffix(token, "}}")
}

func tryExpand(component Component) (string, error) {
	tokens := strings.Split(component.Content, " ")
	output := []string{}

	for _, token := range tokens {
		if isPluginCall(token) {
			callResult, err := RunPlugin(token)
			if err != nil {
				return "", err
			}

			output = append(output, callResult...)
		} else {
			output = append(output, token)
		}
	}

	return strings.Join(output, " "), nil
}

func LoadTemplateIntoCreator(tmpl Template) {
	currentCreatorTemplate = tmpl.Name
	creatorComponents = make([]Component, len(tmpl.Layout))
	copy(creatorComponents, tmpl.Layout)
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

		case QRComponent:
			contentEntry := widget.NewEntry()
			contentEntry.SetText(c.Content)
			idx := i
			contentEntry.OnChanged = func(s string) {
				creatorComponents[idx].Content = s
			}

			creatorContainer.Add(container.NewVBox(
				widget.NewLabel(c.Name),
				contentEntry,
			))
		case ImageComponent:
			idx := i
			var pickBtn *widget.Button
			pickBtn = widget.NewButton("Choose Image", func() {
				fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
					if err != nil || reader == nil {
						return
					}
					defer reader.Close()

					buf := new(bytes.Buffer)
					_, err = io.Copy(buf, reader)
					if err != nil {
						dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
						return
					}

					creatorComponents[idx].Content = base64.StdEncoding.EncodeToString(buf.Bytes())

					img := canvas.NewImageFromReader(bytes.NewReader(buf.Bytes()), reader.URI().Name())
					img.FillMode = canvas.ImageFillContain
					img.SetMinSize(fyne.NewSize(200, 150))

					creatorContainer.Objects[idx] = container.NewVBox(
						widget.NewLabel(c.Name),
						img,
						pickBtn,
					)
					creatorContainer.Refresh()
				}, fyne.CurrentApp().Driver().AllWindows()[0])
				fd.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg"}))
				fd.Show()
			})

			var preview fyne.CanvasObject
			if c.Content != "" {
				data, err := base64.StdEncoding.DecodeString(c.Content)
				if err == nil {
					preview = canvas.NewImageFromReader(bytes.NewReader(data), c.Name)
					preview.(*canvas.Image).FillMode = canvas.ImageFillContain
					preview.(*canvas.Image).SetMinSize(fyne.NewSize(200, 150))
				} else {
					preview = widget.NewLabel("Invalid image data")
				}
			} else {
				preview = widget.NewLabel("No image selected")
			}

			creatorContainer.Add(container.NewVBox(
				widget.NewLabel(c.Name),
				preview,
				pickBtn,
			))
		}
	}
	creatorContainer.Refresh()
}

func CreateUI(w fyne.Window) fyne.CanvasObject {
	creatorContainer = container.NewVBox(widget.NewLabel("No template loaded."))

	templateNameLabel := widget.NewLabel("")
	updateTemplateNameLabel := func() {
		if currentCreatorTemplate != "" {
			templateNameLabel.SetText("Template: " + currentCreatorTemplate)
		} else {
			templateNameLabel.SetText("No template loaded")
		}
	}
	updateTemplateNameLabel()

	loadFromLibraryBtn := widget.NewButton("Load from Library", func() {
		var templateButtons []fyne.CanvasObject
		for _, tmpl := range settings.Library {
			tmplName := tmpl.Name
			btn := widget.NewButton(tmplName, func() {
				LoadTemplateIntoCreator(tmpl)
				updateTemplateNameLabel()
			})
			btn.Importance = widget.MediumImportance
			templateButtons = append(templateButtons, btn)
		}
		buttonList := container.NewVBox(templateButtons...)
		scroll := container.NewVScroll(buttonList)
		scroll.SetMinSize(fyne.NewSize(250, 5*40))
		dialog.ShowCustom("Load Template", "Close", scroll, w)
	})

	loadBtn := widget.NewButton("Load From JSON", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			var imported Template
			jsonParser := json.NewDecoder(reader)
			if err := jsonParser.Decode(&imported); err != nil {
				dialog.ShowError(err, w)
				return
			}

			LoadTemplateIntoCreator(imported)
			updateTemplateNameLabel()
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
			export := Template{
				Name:   currentCreatorTemplate,
				Layout: creatorComponents,
			}
			j, err := json.MarshalIndent(export, "", "  ")
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
		fd.SetFileName(currentCreatorTemplate + ".json")
		fd.Show()
	})

	printBtn := widget.NewButton("Print", func() {
		if len(creatorComponents) == 0 {
			dialog.ShowInformation("No Template", "Load a template first.", w)
			return
		}

		expandedComponents := []Component{}
		for _, component := range creatorComponents {
			if component.Type == TextComponent || component.Type == QRComponent {
				output, err := tryExpand(component)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				component.Content = output
			}

			expandedComponents = append(expandedComponents, component)
		}

		err := SendToPrinter(expandedComponents, settings.PrintServerURL)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Printed", "Receipt has been printed!", w)
		}
	})
	printBtn.Importance = widget.HighImportance

	buttons := container.NewHBox(loadFromLibraryBtn, loadBtn, exportBtn)

	if len(creatorComponents) != 0 && currentCreatorTemplate != "" {
		tmpl := Template{
			Name:   currentCreatorTemplate,
			Layout: creatorComponents,
		}
		LoadTemplateIntoCreator(tmpl)
	}

	return container.NewVBox(
		MakeHeaderLabel("Receipt Creator"),
		templateNameLabel,
		buttons,
		widget.NewSeparator(),
		creatorContainer,
		container.NewVBox(printBtn),
	)
}
