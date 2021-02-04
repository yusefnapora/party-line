package main

import (
	"flag"
	"fmt"
	"github.com/webview/webview"
	"os"

	"github.com/ipfs/go-log"
)

func main() {
	log.SetLogLevel("p2p/hole-punch", "INFO")

	osUser, found := os.LookupEnv("USER")
	if !found {
		osUser = "unknown user"
	}

	port := flag.Int("ui-port", 7777, "port number for backend / frontend comms")
	headless := flag.Bool("headless", false, "don't open a webview on start")
	nick := flag.String("nick", osUser, "nickname / display name")

	flag.Parse()

	// TODO: add "connect to peer id" box to UI. for now, we just pass pids on the command line
	remotePeers := flag.Args()


	a, err := NewApp(*port, *nick)
	if err != nil {
		panic(err)
	}
	a.Start()
	a.ConnectToPeers(remotePeers...)

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
