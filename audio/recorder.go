package audio

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
	"time"

	"github.com/tevino/abool"
)

type Recording struct {
	ID string
	Frames [][]byte
}

type Recorder struct {
	recording abool.AtomicBool
	stateLk sync.Mutex
	recordings map[string]*Recording
	stopCh chan struct{}

	inputDevice *InputDevice

	// TODO: move playback outside of Recorder
	outputDevice *OutputDevice
}

const sampleRate = 48000

func NewRecorder() (*Recorder, error){
	inputDevice, err := OpenInputDevice(sampleRate)
	if err != nil {
		panic(err)
	}

	outputDevice, err := OpenOutputDevice(sampleRate)
	if err != nil {
		return nil, err
	}

	return &Recorder{
		inputDevice: inputDevice,
		outputDevice: outputDevice,
		recordings: make(map[string]*Recording),
	}, nil
}


func (r *Recorder) BeginRecording(maxDuration time.Duration) (string, error) {
	if r.recording.IsSet() {
		return "", fmt.Errorf("recording already in progress")
	}
	r.stateLk.Lock()
	r.recording.Set()
	defer r.stateLk.Unlock()

	id := uuid.New().String()
	rec := &Recording{ID: id}
	r.recordings[id] = rec
	r.stopCh = make(chan struct{})

	if maxDuration != 0 {
		time.AfterFunc(maxDuration, func() {
			r.stopCh <- struct{}{}
		})
	}

	go r.doRecording(rec)

	return id, nil
}

func (r *Recorder) doRecording(rec *Recording) {

	frameCh := r.inputDevice.ReadOpus(r.stopCh)
	for frame := range frameCh {
		rec.Frames = append(rec.Frames, frame)
	}

	r.recording.UnSet()
}

func (r *Recorder) StopRecording() error {
	if r.recording.IsNotSet() {
		return fmt.Errorf("no recording in progress")
	}
	r.stopCh <- struct{}{}
	return nil
}


func (r *Recorder) PlayRecording(id string) error {
	fmt.Printf("Recorder.PlayRecording: %s\n", id)
	r.stateLk.Lock()
	defer r.stateLk.Unlock()
	rec, ok := r.recordings[id]
	if !ok {
		return fmt.Errorf("no recording found with id %s", id)
	}
	return r.outputDevice.PlayRecording(rec)
}