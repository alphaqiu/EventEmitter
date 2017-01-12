package event

import (
	"code.google.com/p/go-uuid/uuid"
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
	UnSubscribe(identity ...string)
	UnSubscribeAll()
	Emit(event ...Event)
}

func NewEmitter(eventBufSize int) Emitter {
	emitter := new(eventEmitter)
	emitter.hub = make(map[EventType]*broadcaster)
	emitter.eventListener = make(chan Event)
	emitter.observer = make(chan subscriber)
	if eventBufSize <= 0 {
		eventBufSize = 256
	}
	emitter.eventBufSize = eventBufSize
	go emitter.dispatch()
	return emitter
}

type eventEmitter struct {
	hub           map[EventType]*broadcaster
	eventListener chan Event
	observer      chan subscriber
	eventBufSize  int
}

func (e *eventEmitter) dispatch() {
	for {
		select {
		case event := <-e.eventListener:
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
					b.emitter = e
					go b.start()
					e.hub[suber.eventName] = b
				}
				b.dealRegister(suber)
			case unSubscribe:
				e.unSubscribe(suber)
			case unSubscribeAll:
				e.unSubscribe(suber)
			}
		}
	}
}

func (e *eventEmitter) unSubscribe(suber subscriber) {
	for _, b := range e.hub {
		go b.dealRegister(suber)
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
		identity:        identity,
		callback:        callback,
		subscribeAction: subscribe,
		eventName:       eventName,
		subscribeType:   subType,
		response:        response,
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

func (e *eventEmitter) UnSubscribe(identities ...string) {
	for _, identity := range identities {
		e.observer <- subscriber{
			identity:        identity,
			subscribeAction: unSubscribe,
		}
	}
}

func (e *eventEmitter) UnSubscribeAll() {
	e.observer <- subscriber{
		subscribeAction: unSubscribeAll,
	}
}

func (e *eventEmitter) Emit(events ...Event) {
	for _, event := range events {
		e.eventListener <- event
	}
}

type subscribeType string

const (
	fireOnce       subscribeType = "fireOnce"
	fireAllways    subscribeType = "fireAllways"
	subscribe      EventType     = "subscribe"
	unSubscribe    EventType     = "unSubscribe"
	unSubscribeAll EventType     = "unSubscribeAll"
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
	subers       map[string]subscriber
	pipeline     chan Event
	observer     chan subscriber
	eventBufSize int
	emitter      *eventEmitter
}

func (b *broadcaster) start() {
	for {
		select {
		case e := <-b.pipeline:
			b.processEvent(e)
		case suber := <-b.observer:
			switch suber.subscribeAction {
			case subscribe:
				b.processSubscribe(suber)
			case unSubscribe:
				b.processUnSubscribe(suber)
			case unSubscribeAll:
				b.processUnSubscribeAll(suber)
			}
		}
	}
}

func (b *broadcaster) processEvent(event Event) {
	for identity, suber := range b.subers {
		if suber.subscribeType == fireOnce {
			delete(b.subers, identity)
			b.checkEmpty(suber.eventName)
		}

		if suber.callback != nil {
			go suber.callback(event)
		} else {
			suber.event <- event
		}

		if suber.subscribeType == fireOnce {
			close(suber.event)
		}
	}
}

func (b *broadcaster) processSubscribe(suber subscriber) {
	if suber.subscribeType == fireAllways {
		suber.event = make(chan Event, b.eventBufSize)
	} else {
		suber.event = make(chan Event)
	}
	suber.response <- suber.event
	close(suber.response)
	b.subers[suber.identity] = suber
}

func (b *broadcaster) processUnSubscribe(suber subscriber) {
	if s, ok := b.subers[suber.identity]; ok {
		delete(b.subers, suber.identity)
		close(s.event)
		b.checkEmpty(s.eventName)
	}
}

func (b *broadcaster) processUnSubscribeAll(suber subscriber) {
	for id, suber := range b.subers {
		if suber.subscribeType == fireAllways && suber.callback == nil {
			close(suber.event)
		}
		delete(b.subers, id)
		b.checkEmpty(suber.eventName)
	}
}

func (b *broadcaster) checkEmpty(eventType EventType) {
	if len(b.subers) == 0 {
		//clean the parent event type
		//will error?
		delete(b.emitter.hub, eventType)
	}
}

func (b *broadcaster) broadcast(event Event) {
	b.pipeline <- event
}

func (b *broadcaster) dealRegister(suber subscriber) {
	b.observer <- suber
}
