package components

import "github.com/maxence-charriere/go-app/v7/pkg/app"

type MessageInputView struct {
	app.Compo

	textContent string

}

func (v *MessageInputView) Render() app.UI {
	return app.Input().
		Placeholder("say something").
		Style("width", "800px").
		OnChange(v.onChange)
}

func (v *MessageInputView) onChange(ctx app.Context, e app.Event) {
	v.textContent = ctx.JSSrc.Get("value").String()
	app.Log("message content: %s", v.textContent)
}