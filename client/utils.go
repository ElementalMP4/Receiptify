package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func MakeHeaderLabel(label string) fyne.CanvasObject {
	text := canvas.NewText(label, theme.Color(theme.ColorNameForeground))
	text.Alignment = fyne.TextAlignCenter
	text.TextStyle = fyne.TextStyle{Bold: true}
	text.TextSize = 28

	separator := canvas.NewLine(theme.Color(theme.ColorNameSeparator))
	separator.StrokeWidth = 2

	return container.NewVBox(
		container.NewCenter(text),
		separator,
	)
}
