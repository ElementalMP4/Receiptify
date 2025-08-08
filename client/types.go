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
	Align     string        `json:"align,omitempty"` // "left", "center", "right"
}

// ComponentWidget represents a rendered component
type ComponentWidget struct {
	Component     Component
	PreviewWidget fyne.CanvasObject
	Widget        fyne.CanvasObject
}

type ComponentType string
