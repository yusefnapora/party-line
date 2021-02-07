// +build wasm
package components

import (
	"github.com/maxence-charriere/go-app/v7/pkg/app"
	"github.com/yusefnapora/party-line/types"
	"sync"
)

type attachmentClickHandler func(attachment *types.Attachment)

type MessageListView struct {
	app.Compo

	localPeerID string

	msgLk    sync.RWMutex
	messages []*types.Message

	onAttachmentClick attachmentClickHandler
}

func (v *MessageListView) Render() app.UI {
	v.msgLk.RLock()
	defer v.msgLk.RUnlock()

	return app.Div().Class("message-list-view").Body(
		app.Range(v.messages).Slice(func(i int) app.UI {
			msg := v.messages[i]
			return &MessageView{
				msg:               msg,
				fromSelf:          msg.Author.PeerId == v.localPeerID,
				onAttachmentClick: v.onAttachmentClick,
			}
		}))
}

func MessageList(localPeer string, messages []*types.Message, onAttachmentClick attachmentClickHandler) *MessageListView {
	return &MessageListView{
		localPeerID:       localPeer,
		messages:          messages,
		onAttachmentClick: onAttachmentClick,
	}
}

func (v *MessageListView) AddMessage(msg *types.Message) {
	v.msgLk.Lock()
	defer v.msgLk.Unlock()
	v.messages = append(v.messages, msg)
	v.Update()
}

type MessageView struct {
	app.Compo

	msg      *types.Message
	fromSelf bool

	onAttachmentClick attachmentClickHandler
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
			UserAvatar(v.msg.Author, 32),

			app.Span().Body(
				app.Text(v.msg.Author.Nickname),
			).Class("author-name"),

			app.Text(v.msg.TextContent),

			app.If(len(v.msg.Attachments) > 0, app.Range(v.msg.Attachments).Slice(func(i int) app.UI {
				a := v.msg.Attachments[i]
				return &MessageAttachmentView{attachment: a, clickHandler: v.onAttachmentClick}
			})))
}

type MessageAttachmentView struct {
	app.Compo
	attachment   *types.Attachment
	clickHandler attachmentClickHandler
}

func (v *MessageAttachmentView) Render() app.UI {
	if v.attachment.Type != types.AttachmentTypeAudioOpus {
		return app.Div()
	}

	return app.Div().
		Body(Icon("fas fa-file-audio")).OnClick(v.onClick)
}

func (v *MessageAttachmentView) onClick(ctx app.Context, e app.Event) {
	if v.clickHandler != nil {
		v.clickHandler(v.attachment)
	}
}
