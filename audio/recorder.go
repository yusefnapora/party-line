package audio

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tevino/abool"
)

type Recording struct {
	ID string
	Frames [][]byte
}

func (r *Recording) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

func RecordingFromJSON(b []byte) (*Recording, error) {
	var r Recording
	err := json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

type Recorder struct {
	recording abool.AtomicBool
	stopCh chan struct{}

	inputDevice *InputDevice

	store *Store
}

const sampleRate = 48000

func NewRecorder(store *Store) (*Recorder, error){
	inputDevice, err := OpenInputDevice(sampleRate)
	if err != nil {
		panic(err)
	}

	return &Recorder{
		inputDevice: inputDevice,
		store: store,
	}, nil
}


func (r *Recorder) BeginRecording(maxDuration time.Duration) (string, error) {
	if r.recording.IsSet() {
		return "", fmt.Errorf("recording already in progress")
	}
	r.recording.Set()

	rec := r.store.NewLocalRecording()
	r.stopCh = make(chan struct{})

	if maxDuration != 0 {
		time.AfterFunc(maxDuration, func() {
			r.stopCh <- struct{}{}
		})
	}

	go r.doRecording(rec)
	return rec.ID, nil
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
