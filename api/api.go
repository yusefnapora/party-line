package api

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/yusefnapora/party-line/audio"
	"github.com/yusefnapora/party-line/types"
	"io"
	"net/http"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wspb"
	"strings"
	"sync"
	"time"
)

type Handler struct {
	pathPrefix string
	localUser  *types.UserInfo

	audioRecorder *audio.Recorder
	audioStore    *audio.Store

	eventCh      <-chan *types.Event
	evtListeners map[string]chan *types.Event
	listenerLk   sync.Mutex

	dispatcher *Dispatcher
}

func NewHandler(pathPrefix string, localUser *types.UserInfo, recorder *audio.Recorder, store *audio.Store, dispatcher *Dispatcher) (*Handler, error) {

	h := &Handler{
		pathPrefix:    pathPrefix,
		localUser:     localUser,
		audioRecorder: recorder,
		audioStore:    store,
		dispatcher:    dispatcher,
		evtListeners:  make(map[string]chan *types.Event),
	}

	h.eventCh = h.dispatcher.AddListener("api-handler")
	go h.fanoutEvents()

	return h, nil
}

func (h *Handler) addEventListener(id string, ch chan *types.Event) {
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
		//fmt.Printf("pushing event to websocket listeners: %v\n", evt)
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

	w.Header().Set("Content-Type", "application/protobuf")

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

	case "/user-info":
		h.ServeUserInfo(w, r)

	case "/connect-to-peer":
		h.ConnectToPeer(w, r)
	}
}

func (h *Handler) ListAudioInputs(w http.ResponseWriter, r *http.Request) {
	devices := audio.ListInputDevices()
	resp := &types.InputDeviceList{Devices: devices}
	buf, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, "Error encoding response", 500)
		return
	}
	if _, err = w.Write(buf); err != nil {
		fmt.Printf("io error: %s\n", err)
	}
}

func (h *Handler) ListPeers(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "coming soon", 501)
}

func (h *Handler) ConnectToPeer(w http.ResponseWriter, r *http.Request) {
	if ensureMethod("POST", w, r) {
		return
	}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("unmarshal error: %s", err), 400)
		return
	}
	req := &types.ConnectToPeerRequest{}
	if err := proto.Unmarshal(buf, req); err != nil {
		writeErrorResponse(w, fmt.Sprintf("error decoding request: %s", err), 400)
		return
	}

	h.dispatcher.ConnectToPeerRequested(req)
	writeEmptyOk(w)
}

func (h *Handler) PublishMessage(w http.ResponseWriter, r *http.Request) {
	if ensureMethod("POST", w, r) {
		return
	}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("unmarshal error: %s", err), 400)
		return
	}
	msg := &types.Message{}
	if err := proto.Unmarshal(buf, msg); err != nil {
		writeErrorResponse(w, fmt.Sprintf("error decoding request: %s", err), 400)
		return
	}

	h.dispatcher.SendMessage(msg)
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
	eventCh := make(chan *types.Event, 100)
	h.addEventListener(listenerID, eventCh)
	fmt.Printf("SubscribeEvents added listener [%s] for websocket updates\n", listenerID)

	go h.websocketPush(c, eventCh)
}

func (h *Handler) websocketPush(ws *websocket.Conn, eventCh chan *types.Event) {
	defer ws.Close(websocket.StatusInternalError, "the sky is falling")

	for evt := range eventCh {
		//fmt.Printf("sending message on websocket: %v\n", evt)

		err := wspb.Write(context.Background(), ws, evt)
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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("io error: %s", err), 400)
	}
	req := types.BeginAudioRecordingRequest{}
	if err := proto.Unmarshal(body, &req); err != nil {
		writeErrorResponse(w, fmt.Sprintf("error decoding request: %s", err), 400)
		return
	}

	fmt.Printf("request: %v\n", req)

	recordingId, err := h.audioRecorder.BeginRecording(time.Duration(0))
	if err != nil {
		writeErrorResponse(w, err.Error(), 500)
		return
	}

	resp := &types.ApiResponse{Resp: &types.ApiResponse_BeginAudioRecording{BeginAudioRecording: &types.BeginAudioRecordingResponse{
		RecordingId: recordingId,
	}}}

	fmt.Printf("response: %v\n", resp)

	buf, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, "Error encoding response", 500)
		return
	}
	w.Write(buf)
}

func (h *Handler) EndRecording(w http.ResponseWriter, r *http.Request) {
	if ensureMethod("post", w, r) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("io error: %s", err), 400)
	}
	req := types.StopAudioRecordingRequest{}
	if err := proto.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("error decoding request: %s", err), 400)
	}

	err = h.audioRecorder.StopRecording()
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("io error: %s", err), 400)
	}
	req := types.PlayAudioRecordingRequest{}
	if err := proto.Unmarshal(body, &req); err != nil {
		writeErrorResponse(w, fmt.Sprintf("error decoding request: %s", err), 400)
	}

	err = h.audioStore.PlayRecording(req.RecordingId)
	if err != nil {
		writeErrorResponse(w, err.Error(), 500)
	}
}

func (h *Handler) ServeUserInfo(w http.ResponseWriter, r *http.Request) {
	buf, err := proto.Marshal(h.localUser)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("marshal error: %d", err), 500)
		return
	}
	if _, err = w.Write(buf); err != nil {
		fmt.Printf("io error: %s\n", err)
	}
}

func writeEmptyOk(w http.ResponseWriter) {
	w.WriteHeader(200)

	resp := &types.ApiResponse{Resp: &types.ApiResponse_Ok{Ok: &types.OkResponse{}}}
	buf, err := proto.Marshal(resp)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding error response object: %s", err.Error()), 500)
	}
	if _, err = w.Write(buf); err != nil {
		fmt.Printf("io error: %s\n", err)
	}
}

func writeErrorResponse(w http.ResponseWriter, msg string, statusCode int) {
	w.WriteHeader(statusCode)

	resp := &types.ApiResponse{Resp: &types.ApiResponse_Error{Error: &types.ErrorResponse{Details: msg}}}
	buf, err := proto.Marshal(resp)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding error response object: %s", err.Error()), 500)
	}
	if _, err = w.Write(buf); err != nil {
		fmt.Printf("io error: %s\n", err)
	}
}

func ensureMethod(method string, w http.ResponseWriter, r *http.Request) (failed bool) {
	if strings.ToUpper(r.Method) == strings.ToUpper(method) {
		return false
	}

	writeErrorResponse(w, fmt.Sprintf("unsupported method %s - use %s instead", r.Method, method), 400)
	return true
}
