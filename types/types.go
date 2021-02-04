package types

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"time"
	pb "github.com/yusefnapora/party-line/p2p/pb"
)

type InputDeviceInfo struct {
	DeviceID  string
	Name      string
	IsDefault bool
}

type InputDeviceList struct {
	Devices []InputDeviceInfo
}

type UserInfo struct {
	PeerID   string
	Nickname string
}

func UserInfoFromPB(info *pb.UserInfo) (*UserInfo, error) {
	pid, err := peer.IDFromBytes(info.PeerId)
	if err != nil {
		return nil, err
	}
	return &UserInfo{
		PeerID:   pid.Pretty(),
		Nickname: info.Nickname,
	}, nil
}

func (i *UserInfo) ToPB() (*pb.UserInfo, error) {
	pid, err := peer.Decode(i.PeerID)
	if err != nil {
		return nil, err
	}
	pidBytes, err := pid.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &pb.UserInfo{
		PeerId:   pidBytes,
		Nickname: i.Nickname,
	}, nil
}

type Message struct {
	Author      UserInfo
	SentAtTime  time.Time
	TextContent string
	Attachments []MessageAttachment
}

func MessageFromPB(msg *pb.Message) (*Message, error) {
	author, err := UserInfoFromPB(msg.Author)
	if err != nil {
		return nil, err
	}

	sentAt := time.Unix(msg.SentAtTimeUnix, 0)
	attachments := make([]MessageAttachment, len(msg.Attachments))
	for i, a := range msg.Attachments {
		attachments[i] = MessageAttachment{
			ID:      a.Id,
			Type:    a.Type,
			Content: a.Content,
		}
	}

	return &Message{
		Author:      *author,
		SentAtTime:  sentAt,
		TextContent: msg.TextContent,
		Attachments: attachments,
	}, nil
}

func (m *Message) ToPB() (*pb.Message, error) {
	author, err := m.Author.ToPB()
	if err != nil {
		return nil, err
	}

	attachments := make([]*pb.Attachment, len(m.Attachments))
	for i, a := range m.Attachments {
		attachments[i] = &pb.Attachment{
			Id:      a.ID,
			Type:    a.Type,
			Content: a.Content,
		}
	}

	return &pb.Message{
		Author:         author,
		SentAtTimeUnix: m.SentAtTime.Unix(),
		TextContent:    m.TextContent,
		Attachments:    attachments,
	}, nil
}

const (
	AttachmentTypeAudioOpus = "audio/opus"
)

type MessageAttachment struct {
	ID      string
	Type    string
	Content []byte
}

type BeginAudioRecordingRequest struct {
	// MaxDuration is a time.ParseDuration compatible string that limits the recording to the given
	// duration. If unset, there is no max, and the recording must be stopped manually with a
	// StopAudioRecordingRequest
	MaxDuration string `json:",omitempty"`
}

type GenericResponse struct {
	// Error is non-empty if something failed.
	Error string `json:",omitempty"`
}

type BeginAudioRecordingResponse struct {
	GenericResponse

	// RecordingID is a unique ID generated by the server to identify the recording.
	// A non-empty ID indicates that the request was successful and the recording has begun.
	RecordingID string
}

// StopAudioRecordingRequest is sent from the frontend to tell us to stop the current audio recording.
type StopAudioRecordingRequest struct {
	// RecordingID is a unique ID generated by the server to identify the recording.
	RecordingID string
}

// PlayAudioRecordingRequest is sent from the frontend when requesting playback of an audio recording.
type PlayAudioRecordingRequest struct {
	RecordingID string
}

type Event struct {
	Timestamp time.Time
	EventType string
	Payload interface{}
}

const (
	EvtUserJoined = "user-joined"
	EvtMsgReceived = "msg-received"
	EvtMsgSent = "msg-sent"
)


// UserJoinedEvent is pushed to the frontend via websockets when we connect to a new peer.
type UserJoinedEvent struct {
	User UserInfo
}

// MessageReceivedEvent is pushed to the frontend via websockets when we get a message from a peer.
type MessageReceivedEvent struct {
	Message Message
}

// MessageSentEvent is pushed to the frontend via websockets when one of our own messages is sent to our peers successfully.
type MessageSentEvent struct {
	Message Message
}
