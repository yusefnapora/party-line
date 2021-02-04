package api

import (
	"encoding/json"
	"fmt"
	"github.com/yusefnapora/party-line/audio"
	"github.com/yusefnapora/party-line/types"
	"net/http"
	"strings"
	"time"
)

type AppState struct {
	localUser types.UserInfo

	received []types.Message
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
		pathPrefix:    pathPrefix,
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

	case "/peers":
		h.ListPeers(w, r)
	}
}

func (h *Handler) ListAudioInputs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	devices := audio.ListInputDevices()

	enc := json.NewEncoder(w)
	err := enc.Encode(types.InputDeviceList{Devices: devices})
	if err != nil {
		http.Error(w, "Error encoding response", 501)
		return
	}
}

func (h *Handler) ListPeers(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "coming soon", 501)
}

func (h *Handler) ConnectToPeer(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "coming soon", 501)
}

func (h *Handler) StartRecording(w http.ResponseWriter, r *http.Request) {
	if ensureMethod("post", w, r) {
		return
	}

	fmt.Printf("StartRecording\n")
	dec := json.NewDecoder(r.Body)
	req := types.BeginAudioRecordingRequest{}
	if err := dec.Decode(&req); err != nil {
		writeErrorResponse(w, fmt.Sprintf("error decoding request: %s", err), 400)
		return
	}

	fmt.Printf("request: %v\n", req)

	recordingId, err := h.audioRecorder.BeginRecording(time.Duration(0))
	var resp types.BeginAudioRecordingResponse
	if err != nil {
		writeErrorResponse(w, err.Error(), 500)
		return
	} else {
		resp = types.BeginAudioRecordingResponse{RecordingID: recordingId}
	}

	fmt.Printf("response: %v\n", resp)

	enc := json.NewEncoder(w)
	err = enc.Encode(resp)
	if err != nil {
		http.Error(w, "Error encoding response", 500)
		return
	}
}

func (h *Handler) EndRecording(w http.ResponseWriter, r *http.Request) {
	if ensureMethod("post", w, r) {
		return
	}

	dec := json.NewDecoder(r.Body)
	req := types.StopAudioRecordingRequest{}
	if err := dec.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %s", err), 400)
	}

	err := h.audioRecorder.StopRecording()
	if err != nil {
		writeErrorResponse(w, err.Error(), 500)
		return
	}

	writeEmptyOk(w)
}

func (h *Handler) PlayRecording(w http.ResponseWriter, r *http.Request) {
	if ensureMethod("post", w, r) {
		return
	}

	dec := json.NewDecoder(r.Body)
	req := types.PlayAudioRecordingRequest{}
	if err := dec.Decode(&req); err != nil {
		writeErrorResponse(w, fmt.Sprintf("error decoding request: %s", err), 400)
	}

	err := h.audioRecorder.PlayRecording(req.RecordingID)
	if err != nil {
		writeErrorResponse(w, err.Error(), 500)
	}
}

func writeEmptyOk(w http.ResponseWriter) {
	writeErrorResponse(w, "", 200)
}

func writeErrorResponse(w http.ResponseWriter, msg string, statusCode int) {
	w.WriteHeader(statusCode)
	resp := types.GenericResponse{Error: msg}
	enc := json.NewEncoder(w)
	err := enc.Encode(resp)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding error response object: %s", err.Error()), 500)
	}
}

func ensureMethod(method string, w http.ResponseWriter, r *http.Request) (failed bool) {
	if strings.ToUpper(r.Method) == strings.ToUpper(method) {
		return false
	}

	writeErrorResponse(w, fmt.Sprintf("unsupported method %s - use %s instead", r.Method, method), 400)
	return true
}
