package p2p

import (
	"context"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/connmgr"
	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/yusefnapora/party-line/api"
	"github.com/yusefnapora/party-line/audio"
	pb "github.com/yusefnapora/party-line/types"
	"sync"
	"time"

	pbio "github.com/gogo/protobuf/io"
)

const protocolID = "/hacks/party-line"
const maxMessageSize = 1 << 20

type PartyLinePeer struct {
	host host.Host

	localUser  *pb.UserInfo
	dispatcher *api.Dispatcher
	audioStore *audio.Store

	// the dispatcher pushes messages here to send out via libp2p
	publishCh <-chan *pb.Message

	incomingMsgCh chan *pb.Message

	fanoutLk sync.Mutex
	fanout   map[string]chan *pb.Message
}

func NewPeer(dispatcher *api.Dispatcher, publishCh <-chan *pb.Message, audioStore *audio.Store, userNick string, blockLAN bool) (*PartyLinePeer, error) {
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
	opts := []libp2p.Option{
		libp2p.ForceReachabilityPrivate(), libp2p.EnableAutoRelay(),
		libp2p.StaticRelays(relayInfo), libp2p.EnableHolePunching(),
		libp2p.Transport(tcp.NewTCPTransport),
		//libp2p.Transport(quic.NewTransport),
		libp2p.ListenAddrs(ma.StringCast("/ip4/0.0.0.0/tcp/0"), ma.StringCast("/ip4/0.0.0.0/udp/0/quic")),
	}

	if blockLAN {
		opts = append(opts, libp2p.ConnectionGater(&gater{}))
	}

	h, err := libp2p.New(ctx, opts...)

	peer := &PartyLinePeer{
		publishCh:     publishCh,
		dispatcher:    dispatcher,
		audioStore:    audioStore,
		fanout:        make(map[string]chan *pb.Message),
		incomingMsgCh: make(chan *pb.Message, 1024),
	}

	peer.localUser = &pb.UserInfo{
		PeerId:   h.ID().Pretty(),
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

	peer.host = routedhost.Wrap(h, d)

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
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

	var remoteUser *pb.UserInfo
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
	pubCh := p.addFanoutListener(remoteUser.PeerId)

	// push any outgoing messages to the stream
	for msg := range pubCh {
		//fmt.Printf("writing outgoing message to stream: %v\n", msg)
		if err := w.WriteMsg(msg); err != nil {
			fmt.Printf("error publishing message: %s\n", err)
		}
	}
}

func (p *PartyLinePeer) readHello(r pbio.Reader) (*pb.Hello, *pb.UserInfo, error) {
	var hello pb.Hello
	if err := r.ReadMsg(&hello); err != nil {
		fmt.Printf("error reading hello message: %s\n", err)
		return nil, nil, err
	}

	p.dispatcher.PeerJoined(hello.User)

	return &hello, hello.User, nil
}

func (p *PartyLinePeer) sayHello(w pbio.Writer) error {
	hello := &pb.Hello{User: p.localUser}
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
		p.inlineAttachmentContent(msg)

		p.fanoutLk.Lock()
		for _, ch := range p.fanout {
			ch <- msg
		}
		p.fanoutLk.Unlock()
	}
}

func (p *PartyLinePeer) incomingMsgLoop() {
	for msg := range p.incomingMsgCh {
		//fmt.Printf("received message from incoming channel %v\n", pbMsg)

		for _, a := range msg.Attachments {
			rec, err := recordingFromAttachment(a)
			if err != nil {
				fmt.Printf("error unpacking audio recording: %s\n", err)
			} else {
				fmt.Printf("adding audio recording from message to store. recording id: %s\n", rec.ID)
				p.audioStore.AddRecording(rec)
			}
		}

		p.dispatcher.ReceiveMessage(msg)
	}
}

func recordingFromAttachment(a *pb.Attachment) (*audio.Recording, error) {
	switch aa := a.Kind.(type) {
	case *pb.Attachment_Audio:
		rec := audio.Recording{
			ID:     a.Id,
			Frames: aa.Audio.Frames,
		}
		return &rec, nil

	default:
		return nil, fmt.Errorf("unsupported attachment type %T", aa)
	}
}

func (p *PartyLinePeer) inlineAttachmentContent(msg *pb.Message) {
	for _, a := range msg.Attachments {
		switch att := a.Kind.(type) {
		case *pb.Attachment_Audio:

			recording, ok := p.audioStore.GetRecording(a.Id)
			if !ok {
				fmt.Printf("no recording found with attachment id %s\n", a.Id)
				continue
			}

			att.Audio.Codec = "audio/opus"
			att.Audio.Frames = recording.Frames
		}
	}
}

var _ connmgr.ConnectionGater = (*gater)(nil)

type gater struct {
}

func (g gater) InterceptPeerDial(p peer.ID) (allow bool) {
	return true
}

func (g gater) InterceptAddrDial(id peer.ID, a ma.Multiaddr) (allow bool) {
	if manet.IsIPLoopback(a) || manet.IsPrivateAddr(a) {
		//fmt.Printf("blocking dial to local addr %s for peer %s\n", a, id.Pretty())
		return false
	}
	return true
}

func (g gater) InterceptAccept(multiaddrs network.ConnMultiaddrs) (allow bool) {
	return true
}

func (g gater) InterceptSecured(direction network.Direction, id peer.ID, multiaddrs network.ConnMultiaddrs) (allow bool) {
	return true
}

func (g gater) InterceptUpgraded(conn network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}
