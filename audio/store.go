package audio

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
)

type Store struct {
	sync.RWMutex

	recordings map[string]*Recording

	outputDevice *OutputDevice
}

func NewStore() (*Store, error) {
	// TODO: should probably open the device elsewhere and inject here
	outputDevice, err := OpenOutputDevice(sampleRate)
	if err != nil {
		return nil, err
	}

	return &Store{
		recordings:   make(map[string]*Recording),
		outputDevice: outputDevice,
	}, nil
}

func (s *Store) NewLocalRecording() *Recording {
	s.Lock()
	defer s.Unlock()

	id := uuid.New().String()

	rec := &Recording{ID: id}
	s.recordings[id] = rec
	fmt.Printf("new local recording: %s\n", id)
	return rec
}

func (s *Store) AddRecording(rec *Recording) {
	s.Lock()
	defer s.Unlock()

	s.recordings[rec.ID] = rec
}

func (s *Store) PlayRecording(id string) error {
	rec, ok := s.GetRecording(id)
	if !ok {
		return fmt.Errorf("no recording found with id %s", id)
	}
	return s.outputDevice.PlayRecording(rec)
}

func (s *Store) GetRecording(id string) (*Recording, bool) {
	s.RLock()
	defer s.RUnlock()
	rec, ok := s.recordings[id]
	return rec, ok
}
