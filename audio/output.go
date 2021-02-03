package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/hajimehoshi/oto"
	"gopkg.in/hraban/opus.v2"
	"io"
)

type OutputDevice struct {
	sampleRate int
	otoContext *oto.Context
	player     *oto.Player
	opusDec    *opus.Decoder

	pcmCh chan []byte
}

func OpenOutputDevice(sampleRate int) (*OutputDevice, error) {
	const numChannels = 1
	otoCtx, err := oto.NewContext(sampleRate, numChannels, 2, 4096)
	if err != nil {
		return nil, err
	}

	opusDec, err := opus.NewDecoder(sampleRate, 1)
	if err != nil {
		return nil, err
	}

	dev := OutputDevice{
		sampleRate: sampleRate,
		otoContext: otoCtx,
		player:     otoCtx.NewPlayer(),
		opusDec:    opusDec,
		pcmCh:      make(chan []byte, 1000),
	}
	return &dev, nil
}

func (output *OutputDevice) Close() error {
	if err := output.player.Close(); err != nil {
		return err
	}
	return output.otoContext.Close()
}

func (output *OutputDevice) PlayOpus(data []byte) error {
	var frameSizeMs float32 = 20  // if you don't know, go with 60 ms.
	frameSize := frameSizeMs * float32(output.sampleRate) / 1000
	pcm := make([]int16, int(frameSize))
	n, err := output.opusDec.Decode(data, pcm)
	if err != nil {
		return err
	}

	buf := make([]byte, n*2))
	for i := 0; i < n; i++ {
		sample := pcm[i]
		_ = binary.Write(buf, binary.LittleEndian, sample)
	}
	fmt.Printf("playing %d samples (%d bytes pcm)\n", n, buf.Len())
	_, err = io.Copy(output.player, buf)
	return err
}
