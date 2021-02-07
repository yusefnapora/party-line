package components

import "github.com/maxence-charriere/go-app/v7/pkg/app"

type IconView struct {
	app.Compo

	classes []string

	color string
}

func Icon(classes ...string) *IconView {
	return &IconView{classes: classes}
}

func (v *IconView) Color(c string) *IconView {
	v.color = c
	return v
}

func (v *IconView) Render() app.UI {
	tag := app.Span()

	for _, cls := range v.classes {
		tag.Class(cls)
	}
	if v.color != "" {
		tag.Style("color", v.color)
	}

	return tag
}
