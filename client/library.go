package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func LibraryUI(w fyne.Window) fyne.CanvasObject {
	listContainer := container.NewVBox()

	var refreshList func()
	refreshList = func() {
		listContainer.Objects = nil
		for i, tmpl := range settings.Library {
			idx := i
			nameBtn := widget.NewButton(tmpl.Name, func() {
				// TODO implement build-from-template (with some autofill stuff too? Today's date etc.)
			})
			nameBtn.Alignment = widget.ButtonAlignLeading

			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				dialog.ShowConfirm("Delete Template", "Are you sure you want to delete this template?", func(confirm bool) {
					if confirm {
						settings.Library = append(settings.Library[:idx], settings.Library[idx+1:]...)
						saveSettings()
						refreshList()
					}
				}, w)
			})

			row := container.NewBorder(nil, nil, nil, deleteBtn, nameBtn)
			listContainer.Add(row)
		}
		listContainer.Refresh()
	}

	refreshList()

	return container.NewVBox(
		widget.NewLabelWithStyle("Template Library", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		listContainer,
	)
}
