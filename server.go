package main

import (
	"fmt"
	"github.com/yusefnapora/party-line/api"
	"log"
	"net/http"

	"github.com/maxence-charriere/go-app/v7/pkg/app"
)

// StartServer brings up an HTTP server that serves up the frontend. The API used by the frontend is
// mounted at the /api/ prefix (see api.Handler)
func StartServer(port int) {
	apiHandler, err := api.NewHandler("/api")
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
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
	if err != nil {
		log.Fatal(err)
	}
}
