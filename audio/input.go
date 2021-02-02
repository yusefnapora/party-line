package audio

import (
	"fmt"
	"github.com/pion/mediadevices"
	_ "github.com/pion/mediadevices/pkg/driver/microphone"
	"github.com/pion/mediadevices/pkg/io/audio"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/mediadevices/pkg/wave"
	"gopkg.in/hraban/opus.v2"
)

type InputDevice struct {
	sampleRate int
	opusEnc    *opus.Encoder
	reader     audio.Reader
}

// TODO: allow opening specific device by id
func OpenInputDevice(sampleRate int) (*InputDevice, error) {
	stream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Audio: func(constraints *mediadevices.MediaTrackConstraints) {
			constraints.ChannelCount = prop.Int(1)
			constraints.SampleRate = prop.Int(sampleRate)
		},
	})

	if err != nil {
		return nil, err
	}

	track := stream.GetAudioTracks()[0].(*mediadevices.AudioTrack)
	reader := track.NewReader(false)

	enc, err := opus.NewEncoder(sampleRate, 1, opus.AppVoIP)
	if err != nil {
		return nil, err
	}

	dev := InputDevice{
		sampleRate: sampleRate,
		opusEnc:    enc,
		reader:     reader,
	}
	return &dev, nil
}

func (input *InputDevice) ReadOpus(stopCh <-chan struct{}) <-chan []byte {
	outCh := make(chan []byte, 1000)

	go input.readOpus(stopCh, outCh)

	return outCh
}

func (input *InputDevice) readOpus(stopCh <-chan struct{}, opusFrameCh chan []byte) {
	const bufferSize = 1000

	for {
		select {
		case <-stopCh:
			close(opusFrameCh)
			return
		default:
		}

		chunk, release, err := input.reader.Read()
		if err != nil {
			panic(err)
		}
		fmt.Printf("read chunk: %v\n", chunk.ChunkInfo())

		// convert to int16 and feed into opus encoder
		pcm := getRawAudio(chunk)

		// Check the frame size. You don't need to do this if you trust your input.
		frameSize := len(pcm) // must be interleaved if stereo
		frameSizeMs := float32(frameSize) * 1000 / float32(input.sampleRate)
		switch frameSizeMs {
		case 2.5, 5, 10, 20, 40, 60:
			// Good.
		default:
			panic(fmt.Errorf("Illegal frame size: %d bytes (%f ms)", frameSize, frameSizeMs))
		}

		data := make([]byte, bufferSize)
		n, err := input.opusEnc.Encode(pcm, data)
		if err != nil {
			panic(err)
		}
		data = data[:n] // only the first N bytes are opus data. Just like io.Reader.
		opusFrameCh <- data

		// release original sample chunk
		release()
	}
}

func DevicesAvailable() []mediadevices.MediaDeviceInfo {
	var devices []mediadevices.MediaDeviceInfo
	for _, dev := range mediadevices.EnumerateDevices() {
		if dev.Kind != mediadevices.AudioInput {
			continue
		}
		devices = append(devices, dev)
	}
	return devices
}

func getRawAudio(a wave.Audio) []int16 {
	// TODO: buffer pool?
	l := a.ChunkInfo().Len
	pcm := make([]int16, l)
	for i := 0; i < l; i++ {
		sample := wave.Int16SampleFormat.Convert(a.At(i, 0))
		pcm[i] = int16(sample.Int())
	}

	return pcm
}
