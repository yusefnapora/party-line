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

	evtCh        <-chan types.Event
	evtCancelSub func()

	messageListView *MessageListView

	me types.UserInfo
}

func Root(apiClient *client.Client, me types.UserInfo) *RootView {
	return &RootView{
		apiClient:       apiClient,
		me:              me,
		messageListView: MessageList(me.PeerID, nil),
	}
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

func (v *RootView) handleRemoteEvent(evt types.Event) {
	app.Log("remote event: %v", evt)

	switch evt.EventType {
	case types.EvtMsgReceived:
		fallthrough
	case types.EvtMsgSent:
		m := evt.Payload.(map[string]interface{})
		msg, err := msgFromMap(m)
		if err != nil {
			app.Log("unmarshal error: %s", err)
			return
		}
		v.addMessage(msg)

	case types.EvtUserJoined:
		// TODO
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

func (v *RootView) addMessage(msg *types.Message) {
	v.messageListView.AddMessage(msg)

	// update UI
	//v.Update()
}

func (v *RootView) sendMessage(msg *types.Message) error {
	return v.apiClient.PublishMessage(msg)
}

func (v *RootView) textMessageEntered(content string) {
	msg := types.Message{
		Author:      v.me,
		SentAtTime:  time.Now(),
		TextContent: content,
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

	return app.Div().Body(

		v.messageListView,

		app.Div().Body(
			MessageInput(v.textMessageEntered),
			app.Button().
				Class("recording-button").
				Class(btnClass).
				OnClick(v.onClick).
				Body(Icon("fas fa-microphone").Color("white"))),
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
		v.currentRecordingID = resp.RecordingID
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
	a := types.MessageAttachment{
		ID:      recordingID,
		Type:    types.AttachmentTypeAudioOpus,
		Content: nil, // will be filled in on the server by matching the Recording ID
	}

	msg := types.Message{
		Author:      v.me,
		SentAtTime:  time.Now(),
		Attachments: []types.MessageAttachment{a},
	}

	return v.sendMessage(&msg)
}
