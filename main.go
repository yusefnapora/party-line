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

	const recTime = time.Second * 5
	fmt.Printf("recording for %s\n", recTime)
	stopCh := make(chan struct{})
	opusCh := dev.ReadOpus(stopCh)
	time.AfterFunc(recTime, func() {
		stopCh <- struct{}{}
	})

	var opusFrames [][]byte
	for frame := range opusCh {
		opusFrames = append(opusFrames, frame)
	}

	fmt.Printf("recorded %d opus frames\n", len(opusFrames))
	fmt.Println("playing back")

	for _, frame := range opusFrames {
		err = outDev.PlayOpus(frame)
		if err != nil {
			panic(err)
		}
	}
}