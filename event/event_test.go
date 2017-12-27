package event

import (
	"testing"
	"fmt"
	"sync"
)

const (
	fireTimes = 10
)

func TestOn(t *testing.T) {
	emitter := NewEmitter(2)
	wg := new(sync.WaitGroup)
	wg.Add(fireTimes)

	emitter.On(fireEvent, response(wg, t))

	event := GenericEvent{
		Type: fireEvent,
	}

	for i := 0; i < fireTimes; i++ {
		event.Data = []byte(fmt.Sprintf("fire in the hole %d", i+1))
		emitter.Emit(event)
	}
	t.Log("event emited.")
	wg.Wait()
	t.Log("Testing finished.")
}

func response(wg *sync.WaitGroup, t *testing.T) Callback {
	ch := make(chan Event)
	go func() {
		for e := range ch {
			t.Logf("go data: %s\n", e.GetData())
		}
	}()

	return func(event Event){
		ch <- event
		wg.Done()
	}
}