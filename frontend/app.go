// +build wasm

// The UI is running only on a web browser. Therefore, the build instruction
// above is to compile the code below only when the program is built for the
// WebAssembly (wasm) architecture.

package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/maxence-charriere/go-app/v7/pkg/app"
	"github.com/yusefnapora/party-line/client"
	"github.com/yusefnapora/party-line/frontend/components"
	"github.com/yusefnapora/party-line/types"
	"strings"
)



// The main function is the entry point of the UI. It is where components are
// associated with URL paths and where the UI is started.
func main() {
	// get the localhost addr from javascript so we don't have to hardcode it
	scheme := app.Window().Get("location").Get("protocol").String()
	scheme = strings.TrimRight(scheme, ":")
	host := app.Window().Get("location").Get("host").String()
	url := fmt.Sprintf("%s://%s", scheme, host)
	app.Log("making client with root url %s", url)

	apiClient, err := client.NewClient(url)
	if err != nil {
		panic(err)
	}

	// TODO: generate real peer id server side & send to client via api
	user := types.UserInfo{
		PeerID:  "fake-peer-" + uuid.New().String(),
		Nickname: "yusef",
	}

	root := components.Root(apiClient, user)

	app.Route("/", root) // rootComponent component is associated with URL path "/".
	app.Run()                // Launches the PWA.
}
