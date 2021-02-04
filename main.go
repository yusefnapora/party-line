package main

import (
	"flag"
	"fmt"
	"github.com/webview/webview"
)

const port = 7777

func main() {
	fmt.Printf("starting app server on port %d\n", port)
	go StartServer(port)

	headless := flag.Bool("headless", false, "don't open a webview on start")
	flag.Parse()

	if !*headless {
		debug := true
		w := webview.New(debug)
		defer w.Destroy()
		w.SetTitle("NAT Party")
		w.SetSize(1200, 800, webview.HintNone)
		w.Navigate(fmt.Sprintf("http://localhost:%d", port))
		w.Run()
	} else {
		// block forever, since the server is running in a background routine & we don't want to quit yet
		select {}
	}
}