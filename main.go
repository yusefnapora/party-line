package main

import (
	"fmt"
	"github.com/webview/webview"
)

const openWebview = false
const port = 7777

func main() {
	fmt.Printf("starting app server on port %d\n", port)
	go StartServer(port)

	if openWebview {
		debug := true
		w := webview.New(debug)
		defer w.Destroy()
		w.SetTitle("NAT Party")
		w.SetSize(800, 600, webview.HintNone)
		w.Navigate(fmt.Sprintf("http://localhost:%d", port))
		w.Run()
	} else {
		// block forever, since the server is running in a background routine & we don't want to quit yet
		select {}
	}
}