package client

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/gogo/protobuf/proto"
	"github.com/yusefnapora/party-line/types"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wspb"
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

func apiError(fmtStr string, args ...interface{}) error {
	return fmt.Errorf("api error: %s", fmt.Sprintf(fmtStr, args...))
}

func (c *Client) StartAudioRecording() (*types.BeginAudioRecordingResponse, error) {
	req := types.BeginAudioRecordingRequest{}
	body, err := proto.Marshal(&req)
	if err != nil {
		return nil, err
	}

	url := c.apiBaseUrl + "begin-recording"
	resp, err := c.rest.R().EnableTrace().SetBody(body).Post(url)

	if err != nil {
		fmt.Printf("request error: %s\n", err)
		return nil, err
	}

	apiResp := &types.ApiResponse{}
	err = proto.Unmarshal(resp.Body(), apiResp)
	if err != nil {
		fmt.Printf("error decoding api response: %s\n", err)
		return nil, err
	}
	switch r := apiResp.Resp.(type) {
	case *types.ApiResponse_Error:
		return nil, apiError(r.Error.Details)
	case *types.ApiResponse_BeginAudioRecording:
		return r.BeginAudioRecording, nil
	default:
		return nil, apiError("unexpected response type %T", r)
	}
}

func (c *Client) EndAudioRecording(recordingID string) error {
	req := types.StopAudioRecordingRequest{RecordingId: recordingID}
	body, err := proto.Marshal(&req)
	if err != nil {
		return err
	}

	url := c.apiBaseUrl + "end-recording"
	resp, err := c.rest.R().EnableTrace().SetBody(body).Post(url)

	if err != nil {
		return err
	}

	apiResp := &types.ApiResponse{}
	err = proto.Unmarshal(resp.Body(), apiResp)
	if err != nil {
		fmt.Printf("error decoding api response: %s\n", err)
		return err
	}
	switch r := apiResp.Resp.(type) {
	case *types.ApiResponse_Error:
		return apiError(r.Error.Details)
	case *types.ApiResponse_Ok:
		return nil
	default:
		return apiError("unexpected response type %T", r)
	}
}

func (c *Client) PlayAudioRecording(recordingID string) error {
	req := types.PlayAudioRecordingRequest{RecordingId: recordingID}
	body, err := proto.Marshal(&req)
	if err != nil {
		return err
	}

	url := c.apiBaseUrl + "play-recording"
	resp, err := c.rest.R().EnableTrace().SetBody(body).Post(url)

	if err != nil {
		return err
	}

	apiResp := &types.ApiResponse{}
	err = proto.Unmarshal(resp.Body(), apiResp)
	if err != nil {
		fmt.Printf("error decoding api response: %s\n", err)
		return err
	}
	switch r := apiResp.Resp.(type) {
	case *types.ApiResponse_Error:
		return apiError(r.Error.Details)
	case *types.ApiResponse_Ok:
		return nil
	default:
		return apiError("unexpected response type %T", r)
	}
}

func (c *Client) PublishMessage(msg *types.Message) error {
	url := c.apiBaseUrl + "publish-message"
	body, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := c.rest.R().EnableTrace().SetBody(body).Post(url)

	if err != nil {
		return err
	}

	apiResp := &types.ApiResponse{}
	err = proto.Unmarshal(resp.Body(), apiResp)
	if err != nil {
		fmt.Printf("error decoding api response: %s\n", err)
		return err
	}
	switch r := apiResp.Resp.(type) {
	case *types.ApiResponse_Error:
		return apiError(r.Error.Details)
	case *types.ApiResponse_Ok:
		return nil
	default:
		return apiError("unexpected response type %T", r)
	}
}

func (c *Client) GetUserInfo() (*types.UserInfo, error) {
	url := c.apiBaseUrl + "user-info"
	resp, err := c.rest.R().EnableTrace().Get(url)

	if err != nil {
		return nil, err
	}

	var info types.UserInfo
	if err = proto.Unmarshal(resp.Body(), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func removeScheme(url string) string {
	re, err := regexp.Compile("^http(s)?://")
	if err != nil {
		panic(err)
	}
	b := re.ReplaceAll([]byte(url), []byte{})
	return string(b)
}

func (c *Client) SubscribeEvents() (<-chan *types.Event, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	host := removeScheme(c.apiBaseUrl)
	url := fmt.Sprintf("ws://%ssubscribe-events", host)
	fmt.Printf("dialing %s\n", url)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, nil, err
	}
	evtCh := make(chan *types.Event, 1024)
	stopCh := make(chan struct{})
	cancelFn := func() {
		stopCh <- struct{}{}
	}

	go c.readEvents(conn, evtCh, stopCh)
	return evtCh, cancelFn, nil
}

func (c *Client) readEvents(ws *websocket.Conn, evtCh chan *types.Event, stopCh chan struct{}) {
	defer ws.Close(websocket.StatusNormalClosure, "")
	for {
		select {
		case <-stopCh:
			return
		default:
		}

		var evt types.Event
		err := wspb.Read(context.Background(), ws, &evt)
		if err != nil {
			fmt.Printf("error reading message from websocket: %s", err)
			continue
		}
		fmt.Printf("read event from websocket: %v", evt)
		evtCh <- &evt
	}
}
