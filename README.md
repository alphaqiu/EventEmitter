# EventEmitter

[![Build Status](https://travis-ci.org/alphaqiu/EventEmitter.svg?branch=master)](https://travis-ci.org/alphaqiu/EventEmitter) [![Coverage Status](https://coveralls.io/repos/github/alphaqiu/EventEmitter/badge.svg?branch=master)](https://coveralls.io/github/alphaqiu/EventEmitter?branch=master) [![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)](https://www.apache.org/licenses/LICENSE-2.0) [![language](https://img.shields.io/badge/golang-%5E1.5-blue.svg)]()

example：

```go
package main

import (
	"fmt"
	"github.com/alphaqiu/EventEmitter/event"
	"sync"
	"time"
)

const (
	fireEvent event.EventType = "fire"
)

func main() {

	emitter := event.NewEmitter(2)

	c := make(chan string, 1)
	go func() {
		identity, listener := emitter.Subscribe(fireEvent)
		if listener == nil {
			return
		}
		c <- identity

		for event := range listener {
			fmt.Printf("device 1: %s\n", event.GetData())
		}
	}()

	go func() {
		emitter.On(fireEvent, doEvent())
	}()

	go func() {
		_, listener := emitter.Once(fireEvent)
		if listener == nil {
			return
		}

		for event := range listener {
			fmt.Printf("device 2: %s\n", event.GetData())
		}
	}()

	times := 10
	counter := 0
	var wg sync.WaitGroup
	wg.Add(times)
	go func() {
		time.Sleep(time.Second)
		for {
			if counter >= times {
				break
			}

			if counter == 5 {
				emitter.UnSubscribe(<-c)
			}

			time.Sleep(2 * time.Millisecond)
			event := event.GenericEvent{
				EventType: fireEvent,
				Data: []byte(fmt.Sprintf("fire in the hole %d", counter)),
			}
			emitter.Emit(event)
			counter += 1
			wg.Done()
		}
	}()


	wg.Wait()
	time.Sleep(5 * time.Second)
}

func doEvent() event.Callback {
	ch := make(chan event.Event)
	go func() {
		for e := range ch {
			fmt.Printf("device 3: %s\n", e.GetData())
		}
	}()
	return func(event event.Event){
		ch <- event
	}
}
```

