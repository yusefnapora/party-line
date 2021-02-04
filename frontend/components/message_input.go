package components

import "github.com/maxence-charriere/go-app/v7/pkg/app"

type MessageInputView struct {
	app.Compo

	textContent string

	onSubmit func(string)
}

func MessageInput(onSubmit func(string)) *MessageInputView {
	return &MessageInputView{
		onSubmit: onSubmit,
	}
}

func (v *MessageInputView) Render() app.UI {
	return app.Input().
		Placeholder("say something").
		Style("width", "800px").
		OnChange(v.onChange)
}

func (v *MessageInputView) onChange(ctx app.Context, e app.Event) {
	text := ctx.JSSrc.Get("value").String()
	v.onSubmit(text)

	// clear input text
	ctx.JSSrc.Set("value", "")
}
