package main

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/maxence-charriere/go-app/v7/pkg/app"
	"github.com/yusefnapora/party-line/api"
	"github.com/yusefnapora/party-line/audio"
	"github.com/yusefnapora/party-line/p2p"
	"github.com/yusefnapora/party-line/types"
	ma "github.com/multiformats/go-multiaddr"

	"log"
	"net/http"
	"strings"
)

type PartyLineApp struct {
	UIServerPort int

	localUser     *types.UserInfo
	audioRecorder *audio.Recorder
	audioStore    *audio.Store

	peer *p2p.PartyLinePeer

	dispatcher *api.Dispatcher
}

type PartyLineAppConfig struct {
	UIPort          int
	UserNick        string
	BlockLocalDials bool
}

func NewApp(cfg PartyLineAppConfig) (*PartyLineApp, error) {
	audioStore, err := audio.NewStore()
	if err != nil {
		return nil, err
	}

	recorder, err := audio.NewRecorder(audioStore)
	if err != nil {
		return nil, err
	}

	publishCh := make(chan *types.Message, 1024)
	dispatcher := api.NewDispatcher(publishCh)
	peer, err := p2p.NewPeer(dispatcher, publishCh, audioStore, cfg.UserNick, cfg.BlockLocalDials)
	if err != nil {
		return nil, err
	}

	localUser := &types.UserInfo{
		PeerId:   peer.PeerID().Pretty(),
		Nickname: cfg.UserNick,
	}

	a := &PartyLineApp{
		UIServerPort:  cfg.UIPort,
		localUser:     localUser,
		audioRecorder: recorder,
		audioStore:    audioStore,
		dispatcher:    dispatcher,
		peer:          peer,
	}
	return a, nil
}

func (a *PartyLineApp) Start() {
	go a.startUIServer()
}

func (a *PartyLineApp) ConnectToPeers(pidStrs ...string) {
	for _, p := range pidStrs {

		var pid peer.ID
		// treat peer id as a p2p multiadr if it starts with a slash
		if strings.HasPrefix(p, "/") {
			fmt.Printf("parsing addr str: %s\n", p)
			maddr := ma.StringCast(p)
			fmt.Printf("peer multiaddr: %s\n", maddr.String())
			pidPtr, err := a.peer.AddPeerAddr(maddr)
			if err != nil {
				fmt.Printf("error adding peer addr: %s\n", err)
			}
			pid = *pidPtr
		} else {
			var err error
			pid, err = peer.Decode(p)
			if err != nil {
				fmt.Printf("error parsing peer id: %s\n", err)
				continue
			}
		}

		if err := a.peer.ConnectToPeer(pid); err != nil {
			fmt.Printf("connection error: %s\n", err)
		}
	}
}

func (a *PartyLineApp) startUIServer() {
	fmt.Printf("starting UI server on localhost:%d\n", a.UIServerPort)
	apiHandler, err := api.NewHandler("/api", a.localUser, a.audioRecorder, a.audioStore, a.dispatcher)
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
			`<script>
    			window.jdenticon_config = {replaceMode: "observe"};
			</script>
			<script src="https://cdn.jsdelivr.net/npm/jdenticon@3.1.0/dist/jdenticon.min.js"
				integrity="sha384-VngWWnG9GS4jDgsGEUNaoRQtfBGiIKZTiXwm9KpgAeaRn6Y/1tAFiyXqSzqC8Ga/" crossorigin="anonymous">
			</script>`,
		},
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", a.UIServerPort), mux)
	if err != nil {
		log.Fatal(err)
	}
}
