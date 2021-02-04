// +build wasm

// The UI is running only on a web browser. Therefore, the build instruction
// above is to compile the code below only when the program is built for the
// WebAssembly (wasm) architecture.

package main

import (
	"fmt"
	"github.com/maxence-charriere/go-app/v7/pkg/app"
	"github.com/yusefnapora/party-line/client"
)

// hello is a component that displays a simple "Hello World!". A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type hello struct {
	app.Compo

	apiClient *client.Client

	isRecording bool
	currentRecordingID string

}

// The Render method is where the component appearance is defined. Here, a
// "Hello World!" is displayed as a heading.
func (h *hello) Render() app.UI {
	return app.Div().Body(
		app.Button().
			Body(
				app.If(!h.isRecording, app.Text("Record")).
					Else(app.Text("Stop"))).
			Name("Record").OnClick(h.onClick))
}

func (h *hello) onClick(ctx app.Context, e app.Event) {
	app.Log("button clicked")

	if !h.isRecording {
		resp, err := h.apiClient.StartAudioRecording()
		if err != nil {
			app.Log("error starting recording: %s", err)
			return
		}

		h.isRecording = true
		h.currentRecordingID = resp.RecordingID
	} else {
		err := h.apiClient.EndAudioRecording(h.currentRecordingID)
		if err != nil {
			app.Log("error stopping recording: %s", err)
			return
		}

		h.isRecording = false
		h.currentRecordingID = ""
	}

	h.Update()
}


// The main function is the entry point of the UI. It is where components are
// associated with URL paths and where the UI is started.
func main() {
	const port = 7777
	client, err := client.NewClient(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		panic(err)
	}

	root := &hello{
		apiClient: client,
	}

	app.Route("/", root) // hello component is associated with URL path "/".
	app.Run()                // Launches the PWA.
}
