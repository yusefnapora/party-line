// +build wasm

// The UI is running only on a web browser. Therefore, the build instruction
// above is to compile the code below only when the program is built for the
// WebAssembly (wasm) architecture.

package main

import (
	"fmt"
	"github.com/maxence-charriere/go-app/v7/pkg/app"
	"github.com/yusefnapora/party-line/client"
	"github.com/yusefnapora/party-line/frontend/components"
)



// The main function is the entry point of the UI. It is where components are
// associated with URL paths and where the UI is started.
func main() {
	const port = 7777
	apiClient, err := client.NewClient(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		panic(err)
	}

	root := components.Root(apiClient)

	app.Route("/", root) // rootComponent component is associated with URL path "/".
	app.Run()                // Launches the PWA.
}
