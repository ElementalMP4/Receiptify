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
				var howDoWeOpen *dialog.CustomDialog
				howDoWeOpen = dialog.NewCustom("Open Template", "Cancel",
					container.NewVBox(
						widget.NewLabel("How would you like to open this template?"),
						widget.NewButton("Open in Template Builder", func() {
							setActive("editor")
							LoadTemplateIntoEditor(tmpl)
							howDoWeOpen.Dismiss()
						}),
						widget.NewButton("Open in Receipt Creator", func() {
							setActive("create")
							LoadTemplateIntoCreator(tmpl)
							howDoWeOpen.Dismiss()
						}),
					), w,
				)
				howDoWeOpen.Show()
			})
			nameBtn.Alignment = widget.ButtonAlignLeading

			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				dialog.ShowConfirm("Delete Template", "Are you sure you want to delete this template?", func(confirm bool) {
					if confirm {
						settings.Library = append(settings.Library[:idx], settings.Library[idx+1:]...)
						SaveSettings(false, w)
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
		MakeHeaderLabel("Template Library"),
		listContainer,
	)
}
