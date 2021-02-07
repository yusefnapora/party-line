package audio

import (
	"fmt"
	"github.com/hajimehoshi/oto"
	"gopkg.in/hraban/opus.v2"
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
	var frameSizeMs float32 = 20
	frameSize := frameSizeMs * float32(output.sampleRate) / 1000
	pcm := make([]int16, int(frameSize))
	n, err := output.opusDec.Decode(data, pcm)
	if err != nil {
		return err
	}

	buf := make([]byte, n*2)
	for i := 0; i < n; i++ {
		sample := pcm[i]
		buf[i*2] = byte(sample)
		buf[(i*2)+1] = byte(sample >> 8)
	}

	var written = 0
	for written < n*2 {
		written, err = output.player.Write(buf)
		if err != nil {
			return err
		}
		buf = buf[written:]
	}
	return err
}

func (output *OutputDevice) PlayRecording(rec *Recording) error {
	fmt.Printf("playing recording %s\n", rec.ID)
	for _, frame := range rec.Frames {
		if err := output.PlayOpus(frame); err != nil {
			return err
		}
	}
	return nil
}
