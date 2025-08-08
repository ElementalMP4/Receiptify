package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	TextComponent    ComponentType = "text"
	DividerComponent ComponentType = "divider"
)

var components []ComponentWidget
var componentContainer *fyne.Container
var renderedContainer *fyne.Container

func EditorUI(w fyne.Window) fyne.CanvasObject {
	componentContainer = container.NewVBox()
	renderedContainer = container.NewVBox()

	receiptBorder := canvas.NewRectangle(color.White)
	receiptBorder.StrokeColor = color.Gray{Y: 100}
	receiptBorder.StrokeWidth = 2
	receiptWrapper := container.NewStack(receiptBorder, componentContainer)
	receiptWrapper.Resize(fyne.NewSize(300, 400))

	renderBorder := canvas.NewRectangle(color.White)
	renderBorder.StrokeColor = color.Gray{Y: 100}
	renderBorder.StrokeWidth = 2
	renderWrapper := container.NewStack(renderBorder, renderedContainer)
	renderWrapper.Resize(fyne.NewSize(300, 400))

	receiptScroll := container.NewVScroll(receiptWrapper)
	renderScroll := container.NewVScroll(renderWrapper)

	receiptScroll.SetMinSize(fyne.NewSize(320, 400))
	renderScroll.SetMinSize(fyne.NewSize(320, 400))

	receiptBox := container.NewHBox(
		container.NewVBox(widget.NewLabel("Layout Editor"), receiptScroll),
		layout.NewSpacer(),
		container.NewVBox(widget.NewLabel("Rendered Preview"), renderScroll),
	)

	addTextBtn := widget.NewButton("Add Text", func() {
		c := Component{
			Type:     TextComponent,
			Content:  "Editable text here",
			FontSize: 14,
		}
		addComponent(c)
	})

	addDividerBtn := widget.NewButton("Add Divider", func() {
		c := Component{
			Type:      DividerComponent,
			LineWidth: 2,
		}
		addComponent(c)
	})

	exportBtn := widget.NewButton("Export JSON", func() {
		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			defer writer.Close()

			export := []Component{}
			for _, c := range components {
				export = append(export, c.Component)
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
		fd.Show()
	})

	importBtn := widget.NewButton("Import JSON", func() {
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

			components = nil
			componentContainer.Objects = nil

			for _, c := range imported {
				addComponent(c)
			}
			refreshComponentList()
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
		fd.Show()
	})

	printBtn := widget.NewButton("Print", func() {
		export := []Component{}
		for _, c := range components {
			export = append(export, c.Component)
		}

		j, err := json.MarshalIndent(export, "", "  ")
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		printURL := settings.PrintServerURL
		if printURL == "" {
			dialog.ShowError(errors.New("print server URL not set"), w)
			return
		}

		resp, err := http.Post(printURL+"/print-receipt", "application/json", bytes.NewBuffer(j))
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			dialog.ShowError(fmt.Errorf("print failed: %s", string(body)), w)
			return
		}

		dialog.ShowInformation("Printed", "Receipt sent to printer successfully.", w)
	})

	saveToLibraryBtn := widget.NewButton("Save to Library", func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Template name")

		var promptDialog *dialog.ConfirmDialog

		saveFunc := func(name string) {
			exists := -1
			for i, t := range settings.Library {
				if t.Name == name {
					exists = i
					break
				}
			}
			saveTemplate := func() {
				layout := []Component{}
				for _, c := range components {
					layout = append(layout, c.Component)
				}
				newTemplate := Template{
					Name:   name,
					Layout: layout,
				}
				if exists >= 0 {
					settings.Library[exists] = newTemplate
				} else {
					settings.Library = append(settings.Library, newTemplate)
				}
				saveSettings()
				dialog.ShowInformation("Saved", "Template saved to library.", w)
			}

			if exists >= 0 {
				dialog.ShowConfirm("Overwrite?", "A template with this name already exists. Overwrite?", func(confirm bool) {
					if confirm {
						saveTemplate()
					}
				}, w)
			} else {
				saveTemplate()
			}
		}

		promptDialog = dialog.NewCustomConfirm("Save Template", "Save", "Cancel",
			container.NewVBox(widget.NewLabel("Enter a name for your template:"), nameEntry),
			func(confirm bool) {
				if confirm && nameEntry.Text != "" {
					saveFunc(nameEntry.Text)
				}
			}, w)
		promptDialog.Resize(fyne.NewSize(300, 150))
		promptDialog.Show()
	})

	loadFromLibraryBtn := widget.NewButton("Load from Library", func() {
		var templateButtons []fyne.CanvasObject
		for _, tmpl := range settings.Library {
			tmplName := tmpl.Name
			btn := widget.NewButton(tmplName, func() {
				components = nil
				componentContainer.Objects = nil
				for _, c := range tmpl.Layout {
					addComponent(c)
				}
				refreshComponentList()
			})
			btn.Importance = widget.HighImportance
			btn.Resize(fyne.NewSize(320, 40))
			templateButtons = append(templateButtons, btn)
		}
		dialog.ShowCustom("Load Template", "Close",
			container.NewVScroll(container.NewVBox(templateButtons...)), w)
	})

	buttons := container.NewVBox(
		importBtn,
		addTextBtn,
		addDividerBtn,
		exportBtn,
		printBtn,
		saveToLibraryBtn,
		loadFromLibraryBtn,
	)

	return container.NewVBox(
		receiptBox,
		buttons,
	)
}

func addComponent(c Component) {
	var display fyne.CanvasObject

	index := len(components)

	switch c.Type {
	case TextComponent:
		entry := widget.NewEntry()
		entry.Text = c.Content
		entry.MultiLine = true
		entry.Wrapping = fyne.TextWrapWord
		entry.TextStyle.Bold = c.Bold
		entry.TextStyle.Italic = c.Italic
		entry.Resize(fyne.NewSize(300, 30))
		display = entry
	case DividerComponent:
		line := canvas.NewRectangle(color.Black)
		line.SetMinSize(fyne.NewSize(300, float32(c.LineWidth)))
		display = line
	}

	moveUp := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {
		if index > 0 {
			components[index], components[index-1] = components[index-1], components[index]
			refreshComponentList()
		}
	})

	moveDown := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
		if index < len(components)-1 {
			components[index], components[index+1] = components[index+1], components[index]
			refreshComponentList()
		}
	})

	editBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		showEditDialog(c, &components[index])
	})

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if index >= 0 && index < len(components) {
			components = append(components[:index], components[index+1:]...)
			refreshComponentList()
		}
	})

	row := container.NewBorder(nil, nil, moveUp, moveDown,
		container.NewBorder(nil, nil, nil, container.NewHBox(editBtn, deleteBtn), display),
	)

	wrapper := ComponentWidget{Component: c}
	components = append(components, wrapper)
	componentContainer.Add(row)
	refreshComponentList()
	componentContainer.Refresh()
}

func createAlignedText(text string, fontSize int, style fyne.TextStyle, color color.Color, align string) *canvas.Text {
	txt := canvas.NewText(text, color)
	txt.TextSize = float32(fontSize)
	txt.TextStyle = style

	switch align {
	case "center":
		txt.Alignment = fyne.TextAlignCenter
	case "right":
		txt.Alignment = fyne.TextAlignTrailing
	default:
		txt.Alignment = fyne.TextAlignLeading
	}
	return txt
}

func wrapTextLines(text string, fontSize int, width float32, style fyne.TextStyle, color color.Color, align string) fyne.CanvasObject {
	lines := strings.Split(text, "\n")

	var wrappedLines []fyne.CanvasObject
	for _, line := range lines {
		words := strings.Fields(line)
		if len(words) == 0 {
			empty := canvas.NewText(" ", color)
			empty.TextSize = float32(fontSize)
			empty.TextStyle = style
			wrappedLines = append(wrappedLines, empty)
			continue
		}

		var currentLine string
		for _, word := range words {
			testLine := strings.TrimSpace(currentLine + " " + word)
			txt := canvas.NewText(testLine, color)
			txt.TextSize = float32(fontSize)
			txt.TextStyle = style
			txt.Alignment = fyne.TextAlignLeading
			txt.Refresh()
			txt.Resize(fyne.NewSize(width, txt.MinSize().Height))

			if txt.MinSize().Width > width && currentLine != "" {
				wrappedLines = append(wrappedLines, createAlignedText(currentLine, fontSize, style, color, align))
				currentLine = word
			} else {
				currentLine = testLine
			}
		}
		if currentLine != "" {
			wrappedLines = append(wrappedLines, createAlignedText(currentLine, fontSize, style, color, align))
		}
	}

	return container.NewVBox(wrappedLines...)
}

func refreshComponentList() {
	componentContainer.Objects = nil
	renderedContainer.Objects = nil

	for i := range components {
		c := components[i].Component

		var editorWidget fyne.CanvasObject
		switch c.Type {
		case TextComponent:
			entry := widget.NewEntry()
			entry.Text = c.Content
			entry.MultiLine = true
			entry.Wrapping = fyne.TextWrapWord
			entry.TextStyle.Bold = c.Bold
			entry.TextStyle.Italic = c.Italic
			entry.Resize(fyne.NewSize(300, 30))
			entry.OnChanged = func(s string) {
				components[i].Component.Content = s
				refreshComponentList()
			}
			editorWidget = entry
		case DividerComponent:
			line := canvas.NewRectangle(color.Black)
			line.SetMinSize(fyne.NewSize(300, float32(c.LineWidth)))
			editorWidget = line
		}

		moveUp := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {
			if i > 0 {
				components[i], components[i-1] = components[i-1], components[i]
				refreshComponentList()
			}
		})

		moveDown := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
			if i < len(components)-1 {
				components[i], components[i+1] = components[i+1], components[i]
				refreshComponentList()
			}
		})

		editBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
			showEditDialog(c, &components[i])
		})

		deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			if i >= 0 && i < len(components) {
				components = append(components[:i], components[i+1:]...)
				refreshComponentList()
			}
		})

		row := container.NewPadded(
			container.NewBorder(nil, nil, moveUp, moveDown,
				container.NewBorder(nil, nil, nil, container.NewHBox(editBtn, deleteBtn), editorWidget),
			),
		)

		components[i].Widget = row
		componentContainer.Add(row)

		var preview fyne.CanvasObject
		switch c.Type {
		case TextComponent:
			style := fyne.TextStyle{Bold: c.Bold, Italic: c.Italic}
			preview = wrapTextLines(c.Content, c.FontSize, 300, style, color.Black, c.Align)
		case DividerComponent:
			line := canvas.NewRectangle(color.Black)
			line.SetMinSize(fyne.NewSize(300, float32(c.LineWidth)))
			preview = line
		}
		renderedContainer.Add(preview)
	}

	componentContainer.Refresh()
	renderedContainer.Refresh()
}

func showEditDialog(c Component, wrapper *ComponentWidget) {
	form := &widget.Form{}
	updated := c

	var content fyne.CanvasObject
	var editDialog *dialog.CustomDialog

	switch c.Type {
	case TextComponent:
		textEntry := widget.NewEntry()
		textEntry.SetText(c.Content)

		fontSize := widget.NewEntry()
		fontSize.SetText(strconv.Itoa(c.FontSize))

		bold := widget.NewCheck("Bold", nil)
		bold.SetChecked(c.Bold)

		italic := widget.NewCheck("Italic", nil)
		italic.SetChecked(c.Italic)

		underline := widget.NewCheck("Underline", nil)
		underline.SetChecked(c.Underline)

		alignSelect := widget.NewSelect([]string{"left", "center", "right"}, func(s string) {})
		alignSelect.SetSelected(c.Align)
		form.Append("Alignment", alignSelect)

		form.Append("Text", textEntry)
		form.Append("Font Size", fontSize)
		form.Append("", bold)
		form.Append("", italic)
		form.Append("", underline)

		saveBtn := widget.NewButton("Save", func() {
			fs, err := strconv.Atoi(fontSize.Text)
			if err != nil {
				fs = 14
			}
			updated.Content = textEntry.Text
			updated.FontSize = fs
			updated.Bold = bold.Checked
			updated.Italic = italic.Checked
			updated.Underline = underline.Checked
			updated.Align = alignSelect.Selected

			*wrapper = ComponentWidget{Component: updated}
			refreshComponentList()
			editDialog.Hide()
		})

		content = container.NewVBox(form, saveBtn)
	case DividerComponent:
		lineWidth := widget.NewEntry()
		lineWidth.SetText(strconv.Itoa(c.LineWidth))
		form.Append("Line Width", lineWidth)

		saveBtn := widget.NewButton("Save", func() {
			lw, err := strconv.Atoi(lineWidth.Text)
			if err != nil {
				lw = 1
			}
			updated.LineWidth = lw

			*wrapper = ComponentWidget{Component: updated}
			refreshComponentList()
			editDialog.Hide()
		})

		content = container.NewVBox(form, saveBtn)
	}

	editDialog = dialog.NewCustom("Edit Component", "Cancel", content, fyne.CurrentApp().Driver().AllWindows()[0])
	editDialog.Resize(fyne.NewSize(300, 300))
	editDialog.Show()
}
