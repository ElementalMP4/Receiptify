package main

import "fyne.io/fyne/v2"

const (
	TextComponent    ComponentType = "text"
	DividerComponent ComponentType = "divider"
	QRComponent      ComponentType = "qr"
	MacroComponent   ComponentType = "macro"
	HeaderComponent  ComponentType = "header"
)

type Component struct {
	Type      ComponentType `json:"type"`
	Name      string        `json:"name"`
	Content   string        `json:"content,omitempty"`
	Bold      bool          `json:"bold,omitempty"`
	Italic    bool          `json:"italic,omitempty"`
	Underline bool          `json:"underline,omitempty"`
	FontSize  string        `json:"font_size,omitempty"`
	LineWidth int           `json:"line_width,omitempty"`
	Align     string        `json:"align,omitempty"`
	Fit       bool          `json:"fit,omitempty"`
	Scale     int           `json:"scale,omitempty"`
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
	PluginPath     string     `json:"plugins"`
	Library        []Template `json:"library"`
}

type PluginManifest struct {
	PluginName string         `json:"name"`
	Version    string         `json:"version"`
	Functions  []FunctionInfo `json:"functions"`
}

type FunctionInfo struct {
	Name    string   `json:"name"`
	Params  []string `json:"params"`
	Returns []string `json:"returns"`
}

type ComponentType string
