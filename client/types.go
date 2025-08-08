package main

import "fyne.io/fyne/v2"

type Component struct {
	Type      ComponentType `json:"type"`
	Content   string        `json:"content,omitempty"`
	Bold      bool          `json:"bold,omitempty"`
	Italic    bool          `json:"italic,omitempty"`
	Underline bool          `json:"underline,omitempty"`
	FontSize  int           `json:"font_size,omitempty"`
	LineWidth int           `json:"line_width,omitempty"`
	Align     string        `json:"align,omitempty"`
}

type ComponentWidget struct {
	Component     Component
	PreviewWidget fyne.CanvasObject
	Widget        fyne.CanvasObject
}

type Template struct {
	Name   string      `json:"name"`
	Layout []Component `json:"layout"`
}

type AppSettings struct {
	PrintServerURL string     `json:"print_server_url"`
	Library        []Template `json:"library"`
}

type ComponentType string
