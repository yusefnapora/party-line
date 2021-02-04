package main

import (
	"flag"
	"fmt"
	"github.com/webview/webview"
)

func main() {

	port := flag.Int("ui-port", 7777, "port number for backend / frontend comms")
	headless := flag.Bool("headless", false, "don't open a webview on start")
	flag.Parse()

	a, err := NewApp(*port)
	if err != nil {
		panic(err)
	}
	a.Start()

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
