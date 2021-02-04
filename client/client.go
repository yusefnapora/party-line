package client

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/yusefnapora/party-line/types"
)

type Client struct {
	apiBaseUrl string
	rest *resty.Client
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