package api

import (
	"github.com/yusefnapora/party-line/types"
	"sync"
	"time"
)


type Dispatcher struct {
	incoming chan types.Message
	outgoing chan types.Message

	stop       chan struct{}
	listeners  map[string]chan types.Event
	listenerLk sync.Mutex
}

func NewDispatcher() *Dispatcher {
	d := &Dispatcher{
		incoming: make(chan types.Message, 1024),
		outgoing: make(chan types.Message, 1024),
		stop:     make(chan struct{}),
		listeners: make(map[string] chan types.Event),
	}

	go d.loop()

	return d
}

func (d *Dispatcher) SendMessage(msg types.Message) {
	d.outgoing <- msg
}

func (d *Dispatcher) ReceiveMessage(msg types.Message) {
	d.incoming <- msg
}

func (d *Dispatcher) Stop() {
	d.stop <- struct{}{}
}

func (d *Dispatcher) AddListener(id string) <-chan types.Event {
	d.listenerLk.Lock()
	defer d.listenerLk.Unlock()

	ch := make(chan types.Event, 1024)
	d.listeners[id] = ch
	return ch
}

func (d *Dispatcher) RemoveListener(id string) {
	d.listenerLk.Lock()
	defer d.listenerLk.Unlock()
	delete(d.listeners, id)
}

func (d *Dispatcher) loop() {
	for {
		select {
		case <-d.stop:
			return

		case msg := <-d.incoming:
			d.handleIncoming(msg)

		case msg := <-d.outgoing:
			d.handleOutgoing(msg)
		}
	}
}

func (d *Dispatcher) handleIncoming(msg types.Message) {
	evt := types.Event{
		Timestamp: time.Now(),
		EventType: types.EvtMsgReceived,
		Payload:   msg,
	}
	d.pushToListeners(evt)
}

func (d *Dispatcher) handleOutgoing(msg types.Message) {
	// TODO: send via libp2p. for now, we just forward to listeners so the user sees their own message
	evt := types.Event{
		Timestamp: time.Now(),
		EventType: types.EvtMsgSent,
		Payload:   msg,
	}
	d.pushToListeners(evt)
}

func (d *Dispatcher) pushToListeners(msg types.Event) {
	d.listenerLk.Lock()
	defer d.listenerLk.Unlock()

	for _, listener := range d.listeners {
		listener <- msg
	}
}