package main

import (
	"bytes"
	"encoding/json"
	"image/color"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var components []ComponentWidget
var componentContainer *fyne.Container
var renderedContainer *fyne.Container

var currentTemplateName string

func LoadTemplateIntoEditor(tmpl Template) {
	components = []ComponentWidget{}
	currentTemplateName = tmpl.Name
	for _, comp := range tmpl.Layout {
		addComponent(comp)
	}
}

func EditorUI(w fyne.Window) fyne.CanvasObject {
	componentContainer = container.NewVBox()
	renderedContainer = container.NewVBox()

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Template name")
	nameEntry.SetText(currentTemplateName)
	nameEntry.OnChanged = func(s string) {
		currentTemplateName = s
	}

	receiptBorder := canvas.NewRectangle(color.White)
	receiptBorder.StrokeColor = color.Gray{Y: 100}
	receiptBorder.StrokeWidth = 2
	receiptWrapper := container.NewStack(receiptBorder, componentContainer)
	receiptWrapper.Resize(fyne.NewSize(300, 400))

	renderBorder := canvas.NewRectangle(color.White)
	renderBorder.StrokeColor = color.Gray{Y: 100}
	renderBorder.StrokeWidth = 2

	paddedContainer := container.NewPadded(renderedContainer)
	renderWrapper := container.NewStack(renderBorder, paddedContainer)
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
			Name:     "Text Component",
			Content:  "Editable text here",
			FontSize: "14",
		}
		addComponent(c)
	})

	addDividerBtn := widget.NewButton("Add Divider", func() {
		c := Component{
			Type:      DividerComponent,
			Name:      "Divider",
			LineWidth: 2,
		}
		addComponent(c)
	})

	addQRBtn := widget.NewButton("Add QR Code", func() {
		c := Component{
			Type:    QRComponent,
			Name:    "QR Code",
			Content: "https://example.com",
			Fit:     true,
			Scale:   100,
			Align:   "center",
		}
		addComponent(c)
	})

	exportBtn := widget.NewButton("Export JSON", func() {
		if strings.TrimSpace(nameEntry.Text) == "" {
			dialog.ShowInformation("Missing Name", "Please enter a template name before exporting.", w)
			return
		}
		currentTemplateName = nameEntry.Text
		export := Template{Name: currentTemplateName}
		for _, c := range components {
			export.Layout = append(export.Layout, c.Component)
		}

		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil || writer == nil {
				return
			}
			defer writer.Close()

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
		fd.SetFileName(currentTemplateName + ".json")
		fd.Show()
	})

	importBtn := widget.NewButton("Import JSON", func() {
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

			components = nil
			componentContainer.Objects = nil

			currentTemplateName = imported.Name
			nameEntry.SetText(imported.Name)
			for _, c := range imported.Layout {
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

		err := SendToPrinter(export, settings.PrintServerURL)

		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Printed", "Receipt has been printed!", w)
		}
	})

	saveToLibraryBtn := widget.NewButton("Save to Library", func() {
		if strings.TrimSpace(nameEntry.Text) == "" {
			dialog.ShowInformation("Missing Name", "Please enter a template name before saving.", w)
			return
		}
		currentTemplateName = nameEntry.Text

		exists := -1
		for i, t := range settings.Library {
			if t.Name == currentTemplateName {
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
				Name:   currentTemplateName,
				Layout: layout,
			}
			if exists >= 0 {
				settings.Library[exists] = newTemplate
			} else {
				settings.Library = append(settings.Library, newTemplate)
			}
			SaveSettings(false, w)
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
	})

	loadFromLibraryBtn := widget.NewButton("Load from Library", func() {
		var templateButtons []fyne.CanvasObject
		for _, tmpl := range settings.Library {
			btn := widget.NewButton(tmpl.Name, func() {
				components = nil
				componentContainer.Objects = nil
				for _, c := range tmpl.Layout {
					addComponent(c)
				}
				currentTemplateName = tmpl.Name
				nameEntry.SetText(tmpl.Name)
				refreshComponentList()
			})
			btn.Importance = widget.MediumImportance
			templateButtons = append(templateButtons, btn)
		}
		buttonList := container.NewVBox(templateButtons...)
		scroll := container.NewVScroll(buttonList)
		scroll.SetMinSize(fyne.NewSize(250, 5*40))
		dialog.ShowCustom("Load Template", "Close", scroll, w)
	})

	clearBtn := widget.NewButton("Clear", func() {
		components = []ComponentWidget{}
		nameEntry.SetText("")
		currentTemplateName = ""
		refreshComponentList()
	})

	contentControls := container.NewVBox(MakeHeaderLabel("Content"), addTextBtn, addDividerBtn, addQRBtn, clearBtn)
	flowControls := container.NewVBox(MakeHeaderLabel("Data"), importBtn, exportBtn, printBtn)
	libraryControls := container.NewVBox(MakeHeaderLabel("Library"), saveToLibraryBtn, loadFromLibraryBtn)

	buttons := container.NewGridWithColumns(3,
		contentControls,
		flowControls,
		libraryControls,
	)

	refreshComponentList()

	return container.NewVBox(
		MakeHeaderLabel("Template Builder"),
		nameEntry,
		receiptBox,
		buttons,
	)
}

func addComponent(c Component) {
	switch c.Type {
	case TextComponent, MacroComponent, HeaderComponent:
		entry := widget.NewEntry()
		entry.Text = c.Content
		entry.MultiLine = true
		entry.Wrapping = fyne.TextWrapWord
		entry.TextStyle.Bold = c.Bold
		entry.TextStyle.Italic = c.Italic
		entry.Resize(fyne.NewSize(300, 30))
	case DividerComponent:
		line := canvas.NewRectangle(color.Black)
		line.SetMinSize(fyne.NewSize(300, float32(c.LineWidth)))
	case QRComponent:
		img := canvas.NewRectangle(color.Gray{Y: 200})
		img.SetMinSize(fyne.NewSize(100, 100))
	}
	wrapper := ComponentWidget{Component: c}
	components = append(components, wrapper)
	refreshComponentList()
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

func renderQRCode(c Component) fyne.CanvasObject {
	img, err := qrcode.New(c.Content, qrcode.Medium)
	if err != nil {
		return canvas.NewText("Invalid QR data", color.RGBA{255, 0, 0, 255})
	}

	baseWidth := float64(300)
	var size int
	if c.Fit {
		size = int(baseWidth)
	} else {
		if c.Scale <= 0 {
			c.Scale = 100
		}
		size = int(baseWidth * float64(c.Scale) / 100.0)
	}

	var buf bytes.Buffer
	_ = img.Write(size, &buf)
	res := canvas.NewImageFromReader(&buf, "qr.png")
	res.FillMode = canvas.ImageFillContain
	res.SetMinSize(fyne.NewSize(float32(size), float32(size)))

	switch c.Align {
	case "center":
		return container.NewHBox(layout.NewSpacer(), res, layout.NewSpacer())
	case "right":
		return container.NewHBox(layout.NewSpacer(), res)
	default:
		return container.NewHBox(res, layout.NewSpacer())
	}
}

func refreshComponentList() {
	componentContainer.Objects = nil
	renderedContainer.Objects = nil

	for i := range components {
		c := components[i].Component

		var editorWidget fyne.CanvasObject
		switch c.Type {
		case TextComponent, HeaderComponent, MacroComponent:
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
		case QRComponent:
			qrLabel := MakeDarkLabel("QR: " + c.Name)
			bg := canvas.NewRectangle(color.RGBA{R: 30, G: 30, B: 30, A: 255})
			editorWidget = container.NewStack(bg, qrLabel)

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
		case TextComponent, HeaderComponent, MacroComponent:
			style := fyne.TextStyle{Bold: c.Bold, Italic: c.Italic}
			fontSizeValue, err := strconv.Atoi(c.FontSize)
			if err != nil {
				fontSizeValue = 14
			}
			preview = wrapTextLines(c.Content, fontSizeValue, 300, style, color.Black, c.Align)
		case DividerComponent:
			line := canvas.NewRectangle(color.Black)
			line.SetMinSize(fyne.NewSize(300, float32(c.LineWidth)))
			preview = line
		case QRComponent:
			preview = renderQRCode(c)
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
	case TextComponent, HeaderComponent, MacroComponent:
		textEntry := widget.NewEntry()
		textEntry.SetText(c.Content)

		nameEntry := widget.NewEntry()
		nameEntry.SetText(c.Name)

		fontSize := widget.NewEntry()
		fontSize.SetText(c.FontSize)

		bold := widget.NewCheck("Bold", nil)
		bold.SetChecked(c.Bold)

		italic := widget.NewCheck("Italic", nil)
		italic.SetChecked(c.Italic)

		underline := widget.NewCheck("Underline", nil)
		underline.SetChecked(c.Underline)

		alignSelect := widget.NewSelect([]string{"left", "center", "right"}, func(s string) {})
		alignSelect.SetSelected(c.Align)

		typeOverrideSelect := widget.NewSelect([]string{"text", "header", "macro"}, func(s string) {})
		typeOverrideSelect.Selected = string(c.Type)

		form.Append("Alignment", alignSelect)
		form.Append("Text", textEntry)
		form.Append("Name", nameEntry)
		form.Append("Font Size", fontSize)
		form.Append("Type Override", typeOverrideSelect)
		form.Append("", bold)
		form.Append("", italic)
		form.Append("", underline)

		saveBtn := widget.NewButton("Save", func() {
			fs := fontSize.Text
			if fontSize.Text != "fit" {
				_, err := strconv.Atoi(fontSize.Text)
				if err != nil {
					fs = "14"
				}
			}
			updated.Content = textEntry.Text
			updated.Name = nameEntry.Text
			updated.FontSize = fs
			updated.Bold = bold.Checked
			updated.Italic = italic.Checked
			updated.Underline = underline.Checked
			updated.Align = alignSelect.Selected
			updated.Type = ComponentType(typeOverrideSelect.Selected)

			*wrapper = ComponentWidget{Component: updated}
			refreshComponentList()
			editDialog.Hide()
		})

		content = container.NewVBox(form, saveBtn)

	case DividerComponent:
		lineWidth := widget.NewEntry()
		lineWidth.SetText(strconv.Itoa(c.LineWidth))

		nameEntry := widget.NewEntry()
		nameEntry.SetText(c.Name)

		form.Append("Line Width", lineWidth)
		form.Append("Name", nameEntry)

		saveBtn := widget.NewButton("Save", func() {
			lw, err := strconv.Atoi(lineWidth.Text)
			if err != nil {
				lw = 1
			}

			updated.LineWidth = lw
			updated.Name = nameEntry.Text

			*wrapper = ComponentWidget{Component: updated}
			refreshComponentList()
			editDialog.Hide()
		})

		content = container.NewVBox(form, saveBtn)

	case QRComponent:
		contentEntry := widget.NewEntry()
		contentEntry.SetText(c.Content)

		nameEntry := widget.NewEntry()
		nameEntry.SetText(c.Name)

		alignSelect := widget.NewSelect([]string{"left", "center", "right"}, func(s string) {})
		alignSelect.SetSelected(c.Align)

		fitCheck := widget.NewCheck("Fit to available space", nil)
		fitCheck.SetChecked(c.Fit)

		scaleEntry := widget.NewEntry()
		scaleEntry.SetText(strconv.Itoa(c.Scale))
		if c.Fit {
			scaleEntry.Disable()
		}
		fitCheck.OnChanged = func(checked bool) {
			if checked {
				scaleEntry.Disable()
			} else {
				scaleEntry.Enable()
			}
		}

		form.Append("Content", contentEntry)
		form.Append("Name", nameEntry)
		form.Append("Alignment", alignSelect)
		form.Append("", fitCheck)
		form.Append("Scale (%)", scaleEntry)

		saveBtn := widget.NewButton("Save", func() {
			updated.Content = contentEntry.Text
			updated.Name = nameEntry.Text
			updated.Align = alignSelect.Selected
			updated.Fit = fitCheck.Checked
			if !fitCheck.Checked {
				scale, err := strconv.Atoi(scaleEntry.Text)
				if err != nil || scale <= 0 {
					scale = 100
				}
				updated.Scale = scale
			}
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
