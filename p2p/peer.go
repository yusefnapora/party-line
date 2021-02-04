package p2p

import (
	"context"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/yusefnapora/party-line/api"
	"github.com/yusefnapora/party-line/audio"
	pb "github.com/yusefnapora/party-line/p2p/pb"
	"github.com/yusefnapora/party-line/types"
	"sync"
	"time"

	pbio "github.com/gogo/protobuf/io"
)

const protocolID = "/hacks/party-line"
const maxMessageSize = 1 << 20

type PartyLinePeer struct {
	localUser  types.UserInfo
	host       host.Host
	dispatcher *api.Dispatcher
	audioStore *audio.Store

	// the dispatcher pushes messages here to send out via libp2p
	publishCh <-chan types.Message

	incomingMsgCh chan *pb.Message

	fanoutLk sync.Mutex
	fanout map[string] chan *pb.Message
}

func NewPeer(dispatcher *api.Dispatcher, publishCh <-chan types.Message, audioStore *audio.Store, userNick string) (*PartyLinePeer, error) {
	relayId, err := peer.Decode("Qma71QQyJN7Sw7gz1cgJ4C66ubHmvKqBasSegKRugM5qo6")
	if err != nil {
		return nil, err
	}
	relayInfo := []peer.AddrInfo{
		{
			ID:    relayId,
			Addrs: []ma.Multiaddr{ma.StringCast("/ip4/54.255.209.104/tcp/12001"), ma.StringCast("/ip4/54.255.209.104/udp/12001/quic")},
		},
	}

	fmt.Printf("setting up libp2p host...\n")

	ctx := context.Background()
	h, err := libp2p.New(ctx, libp2p.ForceReachabilityPrivate(), libp2p.EnableAutoRelay(),
		libp2p.StaticRelays(relayInfo), libp2p.EnableHolePunching(),
		libp2p.Transport(tcp.NewTCPTransport),
		//libp2p.Transport(quic.NewTransport),
		libp2p.ListenAddrs(ma.StringCast("/ip4/0.0.0.0/tcp/0"), ma.StringCast("/ip4/0.0.0.0/udp/0/quic")),
	)

	peer := &PartyLinePeer{
		host:       h,
		publishCh:  publishCh,
		dispatcher: dispatcher,
		audioStore: audioStore,
		fanout: make(map[string] chan *pb.Message),
		incomingMsgCh: make(chan *pb.Message, 1024),
	}

	peer.localUser = types.UserInfo{
		PeerID:   h.ID().Pretty(),
		Nickname: userNick,
	}

	h.SetStreamHandler(protocolID, peer.handleIncomingStream)

	go peer.fanoutLoop()
	go peer.incomingMsgLoop()

	sub, err := h.EventBus().Subscribe(new(event.EvtNATDeviceTypeChanged))
	if err != nil {
		panic(err)
	}

	// bootstrap with dht so we can connect to more peers and discover our own addresses.
	d, err := dht.New(ctx, h, dht.Mode(dht.ModeClient), dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...))
	if err != nil {
		panic(err)
	}
	d.Bootstrap(ctx)

	// wait till we have a relay addrs
LOOP:
	for {
		time.Sleep(5 * time.Second)
		addrs := h.Addrs()
		for _, a := range addrs {
			if _, err := a.ValueForProtocol(ma.P_CIRCUIT); err == nil {
				break LOOP
			}
		}
	}

	fmt.Println("\n server peer id is: ", h.ID().Pretty())
	fmt.Println("-----------------------------------------------------------------------------------------------------------------------------------")
	fmt.Println("server addrs are:")

	p2pAddr := ma.StringCast("/p2p/" + h.ID().Pretty())
	for _, a := range h.Addrs() {
		fmt.Println(a.Encapsulate(p2pAddr))
	}
	fmt.Println("-----------------------------------------------------------------------------------------------------------------------------------")

	// get NAT types for TCP & UDP
	const numTransports = 1 // I had to disable quic for stupid compat reasons (using go 1.16 beta). set this to 2 if you re-enable
	for i := 0; i < numTransports; i++ {
		select {
		case ev := <-sub.Out():
			evt := ev.(event.EvtNATDeviceTypeChanged)
			if evt.NatDeviceType == network.NATDeviceTypeCone {
				fmt.Printf("\n your NAT device supports NAT traversal via hole punching for %s connections", evt.TransportProtocol)
			} else {
				fmt.Printf("\n your NAT device does NOT support NAT traversal via hole punching for %s connections", evt.TransportProtocol)
			}

		case <-time.After(60 * time.Second):
			panic(errors.New("could not find NAT type"))
		}
	}
	fmt.Println("\n------------------------------------------------------------------------------------------------------------------------------------")
	fmt.Println("accepting connections now")

	return peer, nil
}

func (p *PartyLinePeer) PeerID() peer.ID {
	return p.host.ID()
}

func (p *PartyLinePeer) ConnectToPeer(pid peer.ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 20)
	defer cancel()
	s, err := p.host.NewStream(ctx, pid, protocolID)
	if err != nil {
		return err
	}

	p.handleStream(s, false)
	return nil
}

func (p *PartyLinePeer) AddPeerAddr(addr ma.Multiaddr) (*peer.ID, error) {
	ai, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return nil, err
	}

	for _, a := range ai.Addrs {
		p.host.Peerstore().AddAddr(ai.ID, a, peerstore.AddressTTL)
	}
	return &ai.ID, nil
}


func (p *PartyLinePeer) handleIncomingStream(s network.Stream) {
	p.handleStream(s, true)
}

func (p *PartyLinePeer) handleStream(s network.Stream, inbound bool) {
	fmt.Printf("new stream with peer %s via %v\n", s.Conn().RemotePeer().Pretty(), s.Conn().RemoteMultiaddr())

	r := pbio.NewDelimitedReader(s, maxMessageSize)
	w := pbio.NewDelimitedWriter(s)

	var remoteUser *types.UserInfo
	var err error

	// inbound conns say hello first
	if inbound {
		_, remoteUser, err = p.readHello(r)
		if err != nil {
			fmt.Printf("error reading hello msg: %s\n", err)
			return
		}

		if err = p.sayHello(w); err != nil {
			fmt.Printf("error saying hello: %s\n", err)
		}
	} else {
		if err := p.sayHello(w); err != nil {
			fmt.Printf("error saying hello: %s\n", err)
		}

		_, remoteUser, err = p.readHello(r)
		if err != nil {
			fmt.Printf("error reading hello msg: %s\n", err)
			return
		}
	}

	// kickoff read loop in background
	go p.readFromStream(r)

	// get a new channel to receive outgoing messages on
	pubCh := p.addFanoutListener(remoteUser.PeerID)

	// push any outgoing messages to the stream
	for msg := range pubCh {
		//fmt.Printf("writing outgoing message to stream: %v\n", msg)
		if err := w.WriteMsg(msg); err != nil {
			fmt.Printf("error publishing message: %s\n", err)
		}
	}
}

func (p *PartyLinePeer) readHello(r pbio.Reader) (*pb.Hello, *types.UserInfo, error) {
	var hello pb.Hello
	if err := r.ReadMsg(&hello); err != nil {
		fmt.Printf("error reading hello message: %s\n", err)
		return nil, nil, err
	}

	userInfo, err := types.UserInfoFromPB(hello.User)
	if err != nil {
		fmt.Printf("error converting from pb: %s\n", err)
		return nil, nil, err
	}
	p.dispatcher.PeerJoined(userInfo)

	return &hello, userInfo, nil
}

func (p *PartyLinePeer) sayHello(w pbio.Writer) error {
	user, err := p.localUser.ToPB()
	if err != nil {
		fmt.Printf("error encoding local user info: %s\n", err)
		return err
	}
	hello := &pb.Hello{User: user}
	return w.WriteMsg(hello)
}

func (p *PartyLinePeer) readFromStream(r pbio.ReadCloser) {
	for {
		var msg pb.Message
		err := r.ReadMsg(&msg)
		if err != nil {
			fmt.Printf("error reading protobuf from stream: %s\n", err)
			fmt.Printf("closing stream due to error\n")
			r.Close()
			return
		}

		fmt.Printf("received message from %s\n", msg.Author.Nickname)
		p.incomingMsgCh <- &msg
	}
}


func (p *PartyLinePeer) addFanoutListener(pidStr string) <-chan *pb.Message {
	p.fanoutLk.Lock()
	defer p.fanoutLk.Unlock()

	ch := make(chan *pb.Message, 1024)
	p.fanout[pidStr] = ch
	return ch
}

func (p *PartyLinePeer) removeFanoutListener(pidStr string) {
	p.fanoutLk.Lock()
	defer p.fanoutLk.Unlock()
	delete(p.fanout, pidStr)
}

func (p *PartyLinePeer) fanoutLoop() {
	for msg := range p.publishCh {
		fmt.Printf("converting msg to pb: %v\n", msg)
		pbMsg, err := msg.ToPB()
		if err != nil {
			fmt.Printf("error converting msg to protobuf: %s", err)
			continue
		}

		p.inlineAttachmentContent(pbMsg)

		p.fanoutLk.Lock()
		for _, ch := range p.fanout {
			ch <- pbMsg
		}
		p.fanoutLk.Unlock()
	}
}

func (p *PartyLinePeer) incomingMsgLoop() {
	for pbMsg := range p.incomingMsgCh {
		//fmt.Printf("received message from incoming channel %v\n", pbMsg)

		msg, err := types.MessageFromPB(pbMsg)
		if err != nil {
			fmt.Printf("error converting message from pb: %s\n", err)
			continue
		}

		for _, a := range msg.Attachments {
			if a.Type == types.AttachmentTypeAudioOpus && len(a.Content) > 0{
				rec, err := audio.RecordingFromJSON(a.Content)
				if err != nil {
					fmt.Printf("error unpacking audio recording: %s\n", err)
				} else {
					fmt.Printf("adding audio recording from message to store. recording id: %s\n", rec.ID)
					p.audioStore.AddRecording(rec)
				}
			}
		}

		p.dispatcher.ReceiveMessage(*msg)
	}
}

func (p *PartyLinePeer) inlineAttachmentContent(msg *pb.Message) {
	for _, a := range msg.Attachments {
		if a.Type != types.AttachmentTypeAudioOpus {
			continue
		}

		recording, ok := p.audioStore.GetRecording(a.Id)
		if !ok {
			fmt.Printf("no recording found with attachment id %s\n", a.Id)
			continue
		}

		// converting the recording to json is ugly, but we're pressed for time :)
		bytes, err := recording.ToJSON()
		if err != nil {
			continue
		}
		a.Content = bytes
	}
}