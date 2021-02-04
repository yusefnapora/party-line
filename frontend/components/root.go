package components

import (
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
}

func Root(apiClient *client.Client) *RootView {
	return &RootView{apiClient: apiClient}
}

func (h *RootView) Render() app.UI {
	me := types.UserInfo{
		PeerID:   "my-peer-id",
		Nickname: "yusef",
	}

	aarsh := types.UserInfo{
		PeerID:   "aarsh-peer-id",
		Nickname: "aarsh",
	}

	dummyMessages := []types.Message{
		{
			Author:      me,
			SentAtTime:  time.Now().Add(time.Minute * -1),
			TextContent: "a fake message",
		},

		{
			Author: aarsh,
			SentAtTime: time.Now(),
			TextContent: "a fake reply",
		},
	}

	btnClass := "state-not-recording"
	if h.isRecording {
		btnClass = "state-recording"
	}

	return app.Div().Body(

		MessageList(me.PeerID, dummyMessages),

		app.Div().Body(
			&MessageInputView{},
			app.Button().
				Class("recording-button").
				Class(btnClass).
				Name("Record").OnClick(h.onClick)),


	)
}

func (h *RootView) onClick(ctx app.Context, e app.Event) {
	app.Log("button clicked")

	recID := ""

	if !h.isRecording {
		resp, err := h.apiClient.StartAudioRecording()
		if err != nil {
			app.Log("error starting recording: %s", err)
			return
		}

		h.isRecording = true
		h.currentRecordingID = resp.RecordingID
	} else {
		recID = h.currentRecordingID
		err := h.apiClient.EndAudioRecording(h.currentRecordingID)
		if err != nil {
			app.Log("error stopping recording: %s", err)
			return
		}

		h.isRecording = false
		h.currentRecordingID = ""

	}

	h.Update()

	// TODO: move playback code elsewhere
	if recID != "" {
		// TODO: move playback elsewhere
		go h.apiClient.PlayAudioRecording(recID)
	}
}
