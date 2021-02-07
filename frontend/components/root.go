package components

import (
	"encoding/json"
	"fmt"
	"github.com/maxence-charriere/go-app/v7/pkg/app"
	"github.com/yusefnapora/party-line/client"
	"github.com/yusefnapora/party-line/types"
	"time"
)

type RootView struct {
	app.Compo

	apiClient *client.Client

	isRecording        bool
	currentRecordingID string

	evtCh        <-chan *types.Event
	evtCancelSub func()

	peerListView    *PeerListView
	messageListView *MessageListView

	me *types.UserInfo
}

func Root(apiClient *client.Client, me *types.UserInfo) *RootView {
	v := &RootView{
		apiClient: apiClient,
		me:        me,
	}
	v.messageListView = MessageList(me.PeerId, nil, v.handleAttachmentClick)
	v.peerListView = PeerList([]*types.UserInfo{me})
	return v
}

func (v *RootView) OnMount(ctx app.Context) {
	fmt.Println("root component mounted")
	var err error
	v.evtCh, v.evtCancelSub, err = v.apiClient.SubscribeEvents()
	if err != nil {
		fmt.Printf("subscription error: %s\n", err)
		return
	}

	go v.readEvents(ctx)
}

func (v *RootView) OnDismount(ctx app.Context) {
	if v.evtCancelSub != nil {
		v.evtCancelSub()
	}
}

func (v *RootView) readEvents(ctx app.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case evt := <-v.evtCh:
			v.handleRemoteEvent(evt)
		}
	}
}

func (v *RootView) handleRemoteEvent(evt *types.Event) {
	app.Log("remote event: %v", evt)

	switch e := evt.Evt.(type) {
	case *types.Event_MessageReceived:
		v.addMessage(e.MessageReceived.Message)
	case *types.Event_MessageSent:
		v.addMessage(e.MessageSent.Message)
	case *types.Event_UserJoined:
		v.userJoined(e.UserJoined.User)
	}
}

func (v *RootView) handleAttachmentClick(a *types.Attachment) {
	app.Log("attachment clicked %v", a)
	if a.Type != types.AttachmentTypeAudioOpus {
		return
	}

	if err := v.apiClient.PlayAudioRecording(a.Id); err != nil {
		app.Log("error playing attachment: %s", err)
	}
}

func msgFromMap(m map[string]interface{}) (*types.Message, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var msg types.Message
	if err = json.Unmarshal(bytes, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func userInfoFromMap(m map[string]interface{}) (*types.UserInfo, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var msg types.UserInfo
	if err = json.Unmarshal(bytes, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (v *RootView) userJoined(info *types.UserInfo) {
	app.Log("got user joined event: %v", info)
	v.peerListView.AddUser(info)
}

func (v *RootView) addMessage(msg *types.Message) {
	v.messageListView.AddMessage(msg)
}

func (v *RootView) sendMessage(msg *types.Message) error {
	return v.apiClient.PublishMessage(msg)
}

func (v *RootView) textMessageEntered(content string) {
	msg := types.Message{
		Author:         v.me,
		SentAtTimeUnix: time.Now().Unix(),
		TextContent:    content,
	}
	if err := v.sendMessage(&msg); err != nil {
		fmt.Printf("send error: %s\n", err)
	}
	app.Log("sent message %v", msg)
}

func (v *RootView) Render() app.UI {
	btnClass := "state-not-recording"
	if v.isRecording {
		btnClass = "state-recording"
	}

	return app.Div().Class("root-view").Body(

		app.Div().Class("message-view-container").Body(

			v.messageListView,

			app.Div().Body(
				MessageInput(v.textMessageEntered),
				app.Button().
					Class("recording-button").
					Class(btnClass).
					OnClick(v.onClick).
					Body(Icon("fas fa-microphone").Color("white"))),
		),

		v.peerListView,
	)
}

func (v *RootView) onClick(ctx app.Context, e app.Event) {
	app.Log("button clicked")

	recID := ""

	if !v.isRecording {
		resp, err := v.apiClient.StartAudioRecording()
		if err != nil {
			app.Log("error starting recording: %s", err)
			return
		}

		v.isRecording = true
		v.currentRecordingID = resp.RecordingId
	} else {
		recID = v.currentRecordingID
		err := v.apiClient.EndAudioRecording(v.currentRecordingID)
		if err != nil {
			app.Log("error stopping recording: %s", err)
			return
		}

		v.isRecording = false
		v.currentRecordingID = ""

	}

	v.Update()

	// TODO: move playback & send code elsewhere?
	if recID != "" {
		go v.apiClient.PlayAudioRecording(recID)
		if err := v.sendAudioMessage(recID); err != nil {
			app.Log("error sending audio message: %s", err)
		}
	}
}

func (v *RootView) sendAudioMessage(recordingID string) error {
	a := &types.Attachment{
		Id:      recordingID,
		Type:    types.AttachmentTypeAudioOpus,
		Content: nil, // will be filled in on the server by matching the Recording ID
	}

	msg := types.Message{
		Author:         v.me,
		SentAtTimeUnix: time.Now().Unix(),
		Attachments:    []*types.Attachment{a},
	}

	return v.sendMessage(&msg)
}
