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

	outDev, err := audio.OpenOutputDevice(sampleRate)
	if err != nil {
		panic(err)
	}

	stopCh := make(chan struct{})
	opusCh := dev.ReadOpus(stopCh)

	time.AfterFunc(time.Second * 5, func() {
		stopCh <- struct{}{}
	})

	var opusFrames [][]byte
	for frame := range opusCh {
		//fmt.Printf("opus frame size: %d\n", len(frame))
		opusFrames = append(opusFrames, frame)
	}

	fmt.Printf("recorded %d opus frames\n", len(opusFrames))

	for _, frame := range opusFrames {
		err = outDev.PlayOpus(frame)
		if err != nil {
			panic(err)
		}
	}
}