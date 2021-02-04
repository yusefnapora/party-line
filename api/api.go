package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/yusefnapora/party-line/audio"
	"github.com/yusefnapora/party-line/types"
	"net/http"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
	"strings"
	"sync"
	"time"
)


type Handler struct {
	pathPrefix string

	audioRecorder *audio.Recorder

	eventCh      <-chan types.Event
	evtListeners map[string]chan types.Event
	listenerLk   sync.Mutex

	dispatcher *Dispatcher
}

func NewHandler(pathPrefix string) (*Handler, error) {
	recorder, err := audio.NewRecorder()
	if err != nil {
		return nil, err
	}

	h := &Handler{
		pathPrefix:    pathPrefix,
		audioRecorder: recorder,
		dispatcher: NewDispatcher(),
		evtListeners: make(map[string] chan types.Event),
	}

	h.eventCh = h.dispatcher.AddListener("api-handler")
	go h.fanoutEvents()

	return h, nil
}

func (h *Handler) addEventListener(id string, ch chan types.Event) {
	h.listenerLk.Lock()
	defer h.listenerLk.Unlock()

	h.evtListeners[id] = ch
}

func (h *Handler) removeEventListener(id string) {
	h.listenerLk.Lock()
	defer h.listenerLk.Unlock()

	delete(h.evtListeners, id)
}

func (h *Handler) fanoutEvents() {
	for evt := range h.eventCh {
		fmt.Printf("pushing event to websocket listeners: %v\n", evt)
		h.listenerLk.Lock()
		for _, listenerCh := range h.evtListeners {
			listenerCh <- evt
		}
		h.listenerLk.Unlock()
	}
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

	case "/subscribe-events":
		h.SubscribeEvents(w, r)

	case "/publish-message":
		h.PublishMessage(w, r)
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

func (h *Handler) PublishMessage(w http.ResponseWriter, r *http.Request) {
	if ensureMethod("POST", w, r) {
		return
	}

	dec := json.NewDecoder(r.Body)
	req := types.Message{}
	if err := dec.Decode(&req); err != nil {
		writeErrorResponse(w, fmt.Sprintf("error decoding request: %s", err), 400)
		return
	}

	// TODO: if message has an audio attachment, get the audio data by recording ID and add it to the message
	h.dispatcher.SendMessage(req)
	writeEmptyOk(w)
}

func (h *Handler) SubscribeEvents(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("websocket error: %s", err), 400)
		return
	}

	fmt.Printf("new websocket conn\n")

	listenerID := uuid.New().String()
	eventCh := make(chan types.Event, 100)
	h.addEventListener(listenerID, eventCh)
	fmt.Printf("SubscribeEvents added listener [%s] for websocket updates\n", listenerID)

	go h.websocketPush(c, eventCh)
}

func (h *Handler) websocketPush(ws *websocket.Conn, eventCh chan types.Event) {
	defer ws.Close(websocket.StatusInternalError, "the sky is falling")

	for evt := range eventCh {
		fmt.Printf("sending message on websocket: %v\n", evt)

		err := wsjson.Write(context.Background(), ws, evt)
		if err != nil {
			fmt.Printf("send error: %s", err)
		}
	}

	ws.Close(websocket.StatusNormalClosure, "")
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
