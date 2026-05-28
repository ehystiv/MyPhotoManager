package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type photoTheme struct{ base fyne.Theme }

func newPhotoTheme() fyne.Theme { return &photoTheme{base: theme.DefaultTheme()} }

func (t *photoTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x22, G: 0x6C, B: 0xE0, A: 0xFF}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 0x22, G: 0x6C, B: 0xE0, A: 0x99}
	}
	return t.base.Color(n, v)
}

func (t *photoTheme) Font(s fyne.TextStyle) fyne.Resource { return t.base.Font(s) }
func (t *photoTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return t.base.Icon(n) }
func (t *photoTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameInnerPadding {
		return t.base.Size(n) + 2
	}
	return t.base.Size(n)
}
