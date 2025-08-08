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

	// Wrap rendered preview with border
	renderBorder := canvas.NewRectangle(color.White)
	renderBorder.StrokeColor = color.Gray{Y: 100}
	renderBorder.StrokeWidth = 2
	renderWrapper := container.NewStack(renderBorder, renderedContainer)
	renderWrapper.Resize(fyne.NewSize(300, 400))

	receiptScroll := container.NewVScroll(receiptWrapper)
	renderScroll := container.NewVScroll(renderWrapper)

	receiptScroll.SetMinSize(fyne.NewSize(320, 400))
	renderScroll.SetMinSize(fyne.NewSize(320, 400))

	// Final layout with both views side by side
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

		// Load PrintServerURL from settings
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

	return container.NewVBox(
		receiptBox,
		container.NewHBox(importBtn, addTextBtn, addDividerBtn, exportBtn, printBtn),
	)
}

// Add a new component to the list
func addComponent(c Component) {
	var display fyne.CanvasObject

	index := len(components)

	switch c.Type {
	case TextComponent:
		entry := widget.NewEntry()
		entry.Text = c.Content
		entry.MultiLine = true // <-- Add this line
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

	// Add move up/down buttons
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

// Refresh the visual list of components
func refreshComponentList() {
	componentContainer.Objects = nil
	renderedContainer.Objects = nil

	for i := range components {
		c := components[i].Component

		// --- Editor View ---
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

		// Add padding around the row
		row := container.NewPadded(
			container.NewBorder(nil, nil, moveUp, moveDown,
				container.NewBorder(nil, nil, nil, container.NewHBox(editBtn, deleteBtn), editorWidget),
			),
		)

		components[i].Widget = row
		componentContainer.Add(row)

		// --- Render Preview ---
		var preview fyne.CanvasObject
		switch c.Type {
		case TextComponent:
			text := canvas.NewText(c.Content, color.Black)
			text.TextSize = float32(c.FontSize)
			text.TextStyle.Bold = c.Bold
			text.TextStyle.Italic = c.Italic

			switch c.Align {
			case "center":
				text.Alignment = fyne.TextAlignCenter
			case "right":
				text.Alignment = fyne.TextAlignTrailing
			default:
				text.Alignment = fyne.TextAlignLeading
			}
			preview = text
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

// Edit a component (properties)
func showEditDialog(c Component, wrapper *ComponentWidget) {
	form := &widget.Form{}
	updated := c // work on a copy

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
