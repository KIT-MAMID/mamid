package master

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestBus(t *testing.T) {
	wg := new(sync.WaitGroup)

	bus := NewBus()

	wg.Add(1)
	go func() {
		bus.Run()
		wg.Done()
	}()

	channel1 := bus.GetNewWriteChannel()
	channel2 := bus.GetNewReadChannel()
	channel3 := bus.GetNewReadChannel()

	channel1 <- 9
	assert.Equal(t, 9, <-channel2)
	assert.Equal(t, 9, <-channel3)

	channel4 := bus.GetNewWriteChannel()
	channel5 := bus.GetNewReadChannel()
	channel4 <- 10
	assert.Equal(t, 10, <-channel2)
	assert.Equal(t, 10, <-channel5)

	bus.Kill()

	wg.Wait()
}
