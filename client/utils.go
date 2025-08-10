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

func MakeDarkLabel(text string) fyne.CanvasObject {
	label := canvas.NewText(text, theme.Color(theme.ColorNameForeground))
	label.Alignment = fyne.TextAlignCenter
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	return container.NewStack(bg, label)
}
