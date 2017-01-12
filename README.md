#EventEmitter

使用举例：

```go
package main

import (
	"fmt"
	"git.oschina.net/octopus/EventEmitter/event"
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

