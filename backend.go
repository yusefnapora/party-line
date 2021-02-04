package main

import (
	"fmt"
	"github.com/maxence-charriere/go-app/v7/pkg/app"
	"github.com/yusefnapora/party-line/api"
	"github.com/yusefnapora/party-line/audio"
	"log"
	"net/http"
)

type PartyLineApp struct {
	UIServerPort int

	audioRecorder *audio.Recorder
	audioStore *audio.Store

	// TODO: libp2p host, etc
}

func NewApp(uiPort int) (*PartyLineApp, error) {
	audioStore, err := audio.NewStore()
	if err != nil {
		return nil, err
	}

	recorder, err := audio.NewRecorder(audioStore)
	if err != nil {
		return nil, err
	}

	a := &PartyLineApp{
		UIServerPort:  uiPort,
		audioRecorder: recorder,
		audioStore:    audioStore,
	}
	return a, nil
}

func (a *PartyLineApp) Start() {
	go a.startUIServer()
}

func (a *PartyLineApp) startUIServer() {
	fmt.Printf("starting UI server on localhost:%d\n", a.UIServerPort)
	apiHandler, err := api.NewHandler("/api", a.audioRecorder, a.audioStore)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)

	// app.Handler is a standard HTTP handler that serves the UI and its
	// resources to make it work in a web browser.
	//
	// It implements the http.Handler interface so it can seamlessly be used
	// with the Go HTTP standard library.
	mux.Handle("/", &app.Handler{
		Name:        "NAT Party",
		Description: "Don't let the NAT block your chat!",
		Styles: []string{
			"/web/css/app.css",
		},
		RawHeaders: []string{
			`<script src="https://kit.fontawesome.com/718ec8aa25.js" crossorigin="anonymous"></script>`,
		},
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", a.UIServerPort), mux)
	if err != nil {
		log.Fatal(err)
	}
}