package main

import (
	"fmt"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/io/audio"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/mediadevices/pkg/wave"
	"gopkg.in/hraban/opus.v2"

	_ "github.com/pion/mediadevices/pkg/driver/microphone"
)

const sampleRate = 48000

func getMicReader() (audio.Reader, error) {
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
	return track.NewReader(false), nil
}

func makeOpusEncoder() (*opus.Encoder, error) {
	const channels = 1 // mono; 2 for stereo

	return opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
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

func readFromMic(reader audio.Reader) error {
	const bufferSize = 1000

	enc, err := makeOpusEncoder()
	if err != nil {
		panic(err)
	}

	for {
		chunk, release, err := reader.Read()
		if err != nil {
			panic(err)
		}
		fmt.Printf("read chunk: %v\n", chunk.ChunkInfo())

		// convert to int16 and feed into opus encoder
		pcm := getRawAudio(chunk)

		// Check the frame size. You don't need to do this if you trust your input.
		frameSize := len(pcm) // must be interleaved if stereo
		frameSizeMs := float32(frameSize)  * 1000 / sampleRate
		switch frameSizeMs {
		case 2.5, 5, 10, 20, 40, 60:
			// Good.
		default:
			return fmt.Errorf("Illegal frame size: %d bytes (%f ms)", frameSize, frameSizeMs)
		}

		data := make([]byte, bufferSize)
		n, err := enc.Encode(pcm, data)
		if err != nil {
			return err
		}
		data = data[:n] // only the first N bytes are opus data. Just like io.Reader.

		fmt.Printf("encoded %fms of audio to opus. output size: %d", frameSizeMs, n)

		release()
	}

	return nil
}

func main() {
	//reader, err := getMicReader()
	//if err != nil {
	//	panic(err)
	//}
	//
	//readFromMic(reader)
}