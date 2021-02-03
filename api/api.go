package api

import (
	"encoding/json"
	"fmt"
	"github.com/yusefnapora/party-line/audio"
	"net/http"
	"strings"
	"time"
)

type AppState struct {
	Me UserInfo
}

type Handler struct {
	pathPrefix string

	audioRecorder *audio.Recorder
}

func NewHandler(pathPrefix string) (*Handler, error) {
	recorder, err := audio.NewRecorder()
	if err != nil {
		return nil, err
	}

	return &Handler{
		pathPrefix: pathPrefix,
		audioRecorder: recorder,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, h.pathPrefix)

	fmt.Printf("serving request for %s\n", path)

	switch path {
	case "/audio-inputs":
		h.ListAudioInputs(w, r)

	case "/begin-recording":
		h.StartRecording(w, r)

	case "/end-recording":
		h.EndRecording(w, r)

	case "/play-recording":
		h.PlayRecording(w, r)
	}
}

func (h *Handler) ListAudioInputs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	devices := audio.ListInputDevices()

	enc := json.NewEncoder(w)
	err := enc.Encode(InputDeviceList{Devices: devices})
	if err != nil {
		http.Error(w, "Error encoding response", 501)
		return
	}
}

func (h *Handler) StartRecording(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("Unsupported method %s - use POST instead", r.Method), 400)
		return
	}

	dec := json.NewDecoder(r.Body)
	req := BeginAudioRecordingRequest{}
	if err := dec.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %s", err), 400)
	}

	recordingId, err := h.audioRecorder.BeginRecording(time.Duration(0))
	var resp BeginAudioRecordingResponse
	if err != nil {
		resp = BeginAudioRecordingResponse{
			GenericResponse: GenericResponse{Error: err.Error()},
		}
	} else {
		resp = BeginAudioRecordingResponse{RecordingID: recordingId}
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(resp)
	if err != nil {
		http.Error(w, "Error encoding response", 501)
		return
	}
}

func (h *Handler) EndRecording(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("Unsupported method %s - use POST instead", r.Method), 400)
		return
	}

	dec := json.NewDecoder(r.Body)
	req := StopAudioRecordingRequest{}
	if err := dec.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %s", err), 400)
	}

	resp := h.audioRecorder.StopRecording()
	enc := json.NewEncoder(w)
	err := enc.Encode(resp)
	if err != nil {
		http.Error(w, "Error encoding response", 501)
		return
	}
}

func (h *Handler) PlayRecording(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("Unsupported method %s - use POST instead", r.Method), 400)
		return
	}

	dec := json.NewDecoder(r.Body)
	req := PlayAudioRecordingRequest{}
	if err := dec.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %s", err), 400)
	}

	err := h.audioRecorder.PlayRecording(req.RecordingID)
	if err != nil {
		http.Error(w, err.Error(), 501)
	}
}