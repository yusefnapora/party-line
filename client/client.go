package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/yusefnapora/party-line/types"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
	"regexp"
	"time"
)

type Client struct {
	apiBaseUrl string
	rest       *resty.Client
}

func NewClient(apiHost string) (*Client, error) {
	apiBase := apiHost + "/api/"
	rest := resty.New()
	rest.SetHeader("Content-Type", "application/json")

	return &Client{
		apiBaseUrl: apiBase,
		rest:       rest,
	}, nil
}

func (c *Client) StartAudioRecording() (*types.BeginAudioRecordingResponse, error) {
	req := types.BeginAudioRecordingRequest{}

	url := c.apiBaseUrl + "begin-recording"
	resp, err := c.rest.R().EnableTrace().
		SetBody(req).
		Post(url)

	if err != nil {
		fmt.Printf("request error: %s\n", err)
		return nil, err
	}

	apiResp := &types.BeginAudioRecordingResponse{}
	err = json.Unmarshal(resp.Body(), apiResp)
	if err != nil {
		fmt.Printf("error decoding api response: %s\n", err)
		return nil, err
	}
	return apiResp, nil
}

func (c *Client) EndAudioRecording(recordingID string) error {
	req := types.StopAudioRecordingRequest{RecordingID: recordingID}

	url := c.apiBaseUrl + "end-recording"
	resp, err := c.rest.R().EnableTrace().
		SetBody(req).
		Post(url)

	if err != nil {
		return err
	}

	apiResp := &types.GenericResponse{}
	err = json.Unmarshal(resp.Body(), apiResp)
	if err != nil {
		return err
	}
	if apiResp.Error != "" {
		return fmt.Errorf("api error: %s", apiResp.Error)
	}
	return nil
}

func (c *Client) PlayAudioRecording(recordingID string) error {
	req := types.PlayAudioRecordingRequest{RecordingID: recordingID}

	url := c.apiBaseUrl + "play-recording"
	resp, err := c.rest.R().EnableTrace().
		SetBody(req).
		Post(url)

	if err != nil {
		return err
	}

	apiResp := &types.GenericResponse{}
	err = json.Unmarshal(resp.Body(), apiResp)
	if err != nil {
		return err
	}
	if apiResp.Error != "" {
		return fmt.Errorf("api error: %s", apiResp.Error)
	}
	return nil
}

func (c *Client) PublishMessage(msg *types.Message) error {
	url := c.apiBaseUrl + "publish-message"
	resp, err := c.rest.R().EnableTrace().
		SetBody(msg).
		Post(url)

	if err != nil {
		return err
	}

	apiResp := &types.GenericResponse{}
	err = json.Unmarshal(resp.Body(), apiResp)
	if err != nil {
		return err
	}
	if apiResp.Error != "" {
		return fmt.Errorf("api error: %s", apiResp.Error)
	}
	return nil
}

func removeScheme(url string) string {
	re, err := regexp.Compile("^http(s)?://")
	if err != nil {
		panic(err)
	}
	b := re.ReplaceAll([]byte(url), []byte{})
	return string(b)
}

func (c *Client) SubscribeEvents() (<-chan types.Event, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	host := removeScheme(c.apiBaseUrl)
	url := fmt.Sprintf("ws://%ssubscribe-events", host)
	fmt.Printf("dialing %s\n", url)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, nil, err
	}
	evtCh := make(chan types.Event, 1024)
	stopCh := make(chan struct{})
	cancelFn := func() {
		stopCh <- struct{}{}
	}

	go c.readEvents(conn, evtCh, stopCh)
	return evtCh, cancelFn, nil
}

func (c *Client) readEvents(ws *websocket.Conn, evtCh chan types.Event, stopCh chan struct{}) {
	defer ws.Close(websocket.StatusNormalClosure, "")
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		var evt types.Event
		err := wsjson.Read(context.Background(), ws, &evt)
		if err != nil {
			fmt.Printf("error reading message from websocket: %s", err)
			continue
		}
		fmt.Printf("read event from websocket: %v", evt)
		evtCh <- evt
	}
}
