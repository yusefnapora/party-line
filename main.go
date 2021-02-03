package main

import (
	"fmt"
)

const sampleRate = 48000


func main() {

	const port = 7777
	fmt.Printf("starting app server on port %d\n", port)
	StartServer(port)
}