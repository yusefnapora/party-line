// +build wasm
package components

import (
	"github.com/maxence-charriere/go-app/v7/pkg/app"
	"github.com/yusefnapora/party-line/types"
	"sync"
)

type MessageListView struct {
	app.Compo

	localPeerID string

	msgLk    sync.RWMutex
	messages []types.Message
}

func (v *MessageListView) Render() app.UI {
	v.msgLk.RLock()
	defer v.msgLk.RUnlock()

	return app.Div().Class("message-list-view").Body(
		app.Range(v.messages).Slice(func(i int) app.UI {
			msg := v.messages[i]
			return &MessageView{
				msg:      msg,
				fromSelf: msg.Author.PeerID == v.localPeerID,
			}
		}))
}

func MessageList(localPeer string, messages []types.Message) *MessageListView {
	return &MessageListView{
		localPeerID: localPeer,
		messages:    messages,
	}
}

type MessageView struct {
	app.Compo

	msg      types.Message
	fromSelf bool
}

func (v *MessageView) Render() app.UI {
	msgTypeClass := "peer-message"
	if v.fromSelf {
		msgTypeClass = "self-message"
	}

	return app.Div().
		Class("message-bubble").
		Class(msgTypeClass).
		Body(
			app.Span().Body(
				app.Text(v.msg.Author.Nickname),
			).Class("author-name"),

			app.Text(v.msg.TextContent),
		)
}
