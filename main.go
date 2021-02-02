package main

import (
	"fmt"
	"github.com/yusefnapora/party-line/audio"
	"time"
)

const sampleRate = 48000


func main() {
	dev, err := audio.OpenInputDevice(sampleRate)
	if err != nil {
		panic(err)
	}

	stopCh := make(chan struct{})
	opusCh := dev.ReadOpus(stopCh)

	time.AfterFunc(time.Second * 10, func() {
		stopCh <- struct{}{}
	})

	for frame := range opusCh {
		fmt.Printf("opus frame size: %d\n", len(frame))
	}
}