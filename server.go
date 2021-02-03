package main

import (
	"fmt"
	"github.com/yusefnapora/party-line/api"
	"log"
	"net/http"

	"github.com/maxence-charriere/go-app/v7/pkg/app"
)

// The main function is the entry of the server. It is where the HTTP handler
// that serves the UI is defined and where the server is started.
//
// Note that because main.go and app.go are built for different architectures,
// this main() function is not in conflict with the one in
// app.go.
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
		Name:        "Hello",
		Description: "An Hello World! example",
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
	if err != nil {
		log.Fatal(err)
	}
}
