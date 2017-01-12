package utils

import (
	"code.google.com/p/go-uuid/uuid"
	logger "github.com/alecthomas/log4go"
)

type EventType string
type Event interface {
	GetType() EventType
	GetMetaData() []byte
	GetData() []byte
}

type GenericEvent struct {
	EventType EventType
	Meta      []byte
	Data      []byte
}

func (e GenericEvent) GetType() EventType {
	return e.EventType
}

func (e GenericEvent) GetMetaData() []byte {
	return e.Meta
}

func (e GenericEvent) GetData() []byte {
	return e.Data
}

type Callback func(event Event)

type Emitter interface {
	On(eventName EventType, callback Callback) (identity string)
	Once(eventName EventType) (identity string, event <-chan Event)
	Subscribe(eventName EventType) (identity string, event <-chan Event)
	UnSubscribe(identity string)
	Emit(event Event)
}

func NewEmitter(eventBufSize int) Emitter {
	emitter := new(eventEmitter)
	emitter.hub = make(map[EventType]*broadcaster)
	emitter.listener = make(chan Event)
	emitter.observer = make(chan subscriber)
	if eventBufSize <= 0 {
		eventBufSize = 256
	}
	emitter.eventBufSize = eventBufSize
	go emitter.dispatch()
	return emitter
}

type eventEmitter struct {
	hub          map[EventType]*broadcaster
	listener     chan Event
	observer     chan subscriber
	eventBufSize int
}

func (e *eventEmitter) dispatch() {
	for {
		select {
		case event := <-e.listener:
			if b, ok := e.hub[event.GetType()]; ok {
				b.broadcast(event)
			}
		case suber := <-e.observer:
			switch suber.subscribeAction {
			case subscribe:
				b, ok := e.hub[suber.eventName]
				if !ok {
					b = new(broadcaster)
					b.subers = make(map[string]subscriber)
					b.pipeline = make(chan Event)
					b.observer = make(chan subscriber)
					b.eventBufSize = e.eventBufSize
					go b.start()
					e.hub[suber.eventName] = b
				}
				b.dealRegister(suber)
			case unSubscribe:
				for _, b := range e.hub {
					go b.dealRegister(suber)
				}
			}
		}
	}
}

func (e *eventEmitter) On(eventName EventType, callback Callback) (identity string) {
	identity, _ = e.doSubscribe(eventName, fireAllways, callback)
	return
}

func (e *eventEmitter) Once(eventName EventType) (string, <-chan Event) {
	return e.doSubscribe(eventName, fireOnce, nil)
}

func (e *eventEmitter) Subscribe(eventName EventType) (string, <-chan Event) {
	return e.doSubscribe(eventName, fireAllways, nil)
}

func (e *eventEmitter) doSubscribe(eventName EventType, subType subscribeType, callback Callback) (string, <-chan Event) {
	identity := uuid.New()
	response := make(chan chan Event)

	e.observer <- subscriber{
		identity: identity,
		callback: callback,
		subscribeAction: subscribe,
		eventName: eventName,
		subscribeType: subType,
		response: response,
	}

	event, ok := <-response
	if !ok {
		return "", nil
	}

	if callback != nil {
		close(event)
		return identity, nil
	} else {
		return identity, event
	}
}

func (e *eventEmitter) UnSubscribe(identity string) {
	e.observer <- subscriber{
		identity: identity,
		subscribeAction: unSubscribe,
	}
}

func (e *eventEmitter) Emit(event Event) {
	e.listener <- event
}

type subscribeType string

const (
	fireOnce subscribeType = "fireOnce"
	fireAllways subscribeType = "fireAllways"
	subscribe EventType = "subscribe"
	unSubscribe EventType = "unSubscribe"
)

type subscriber struct {
	identity        string
	callback        Callback
	eventName       EventType
	subscribeType   subscribeType
	subscribeAction EventType
	response        chan chan Event
	event           chan Event
}

type broadcaster struct {
	subers   map[string]subscriber
	pipeline chan Event
	observer chan subscriber
	eventBufSize int
}

func (b *broadcaster) start() {
	for {
		select {
		case e := <-b.pipeline:
			logger.Debug("Received event: %s", e.GetType())
			for identity, suber := range b.subers {
				if suber.subscribeType == fireOnce {
					delete(b.subers, identity)
				}

				if suber.callback != nil {
					go suber.callback(e)
				} else {
					suber.event <- e
				}

				if suber.subscribeType == fireOnce {
					close(suber.event)
				}
			}
		case suber := <-b.observer:
			switch suber.subscribeAction {
			case subscribe:
				if suber.subscribeType == fireAllways {
					suber.event = make(chan Event, b.eventBufSize)
				} else {
					suber.event = make(chan Event)
				}
				suber.response <- suber.event
				close(suber.response)
				b.subers[suber.identity] = suber
			case unSubscribe:
				if s, ok := b.subers[suber.identity]; ok {
					close(s.event)
				}
				delete(b.subers, suber.identity)
			}
		}
	}
}

func (b *broadcaster) broadcast(event Event) {
	b.pipeline <- event
}

func (b *broadcaster) dealRegister(suber subscriber) {
	b.observer <- suber
}