package master

import (
	"log"
	"reflect"
	"sync"
)

type Bus struct {
	//Channels the user can read from and the bus writes to
	readChannels      []chan interface{}
	readChannelsMutex sync.Mutex

	//Channels the user can write to and the bus reads from
	writeChannels      []chan interface{}
	writeChannelsMutex sync.Mutex

	//Write channel used to send commands to the Bus.Run() goroutine
	internalChannel chan interface{}
}

type killBusMessage struct{}
type interruptBusMessage struct{}

func NewBus() *Bus {
	var b Bus
	b.internalChannel = make(chan interface{}, 1000)
	b.readChannels = []chan interface{}{}
	b.writeChannels = []chan interface{}{b.internalChannel}
	return &b
}

func (b *Bus) GetNewReadChannel() chan interface{} {
	channel := make(chan interface{}, 1000)

	b.readChannelsMutex.Lock()
	b.readChannels = append(b.readChannels, channel)
	b.readChannelsMutex.Unlock()

	return channel
}

func (b *Bus) GetNewWriteChannel() chan interface{} {
	channel := make(chan interface{}, 1000)

	b.writeChannelsMutex.Lock()
	b.writeChannels = append(b.writeChannels, channel)
	b.writeChannelsMutex.Unlock()

	//interrupt bus so that new channel is included in next select
	b.internalChannel <- interruptBusMessage{}

	return channel

}

func (b *Bus) Run() {
	for {
		//Read from write channels
		b.writeChannelsMutex.Lock()
		selects := make([]reflect.SelectCase, len(b.writeChannels))
		for i, channel := range b.writeChannels {
			selects[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(channel)}
		}
		b.writeChannelsMutex.Unlock()

		_, recv, recvOk := reflect.Select(selects)
		if recvOk {
			if _, ok := recv.Interface().(killBusMessage); ok {
				//kill the bus
				return
			}
			if _, ok := recv.Interface().(interruptBusMessage); ok {
				//continue with new select so that new channels are included in select
				continue
			}

			//Send to read channels
			b.readChannelsMutex.Lock()
			for _, channel := range b.readChannels {

				//Only send to bus if channel is not full
				select {
				case channel <- recv.Interface():
				default:
					log.Println("Bus channel full - dropping message")
				}
			}
			b.readChannelsMutex.Unlock()
		}
	}
}

func (b *Bus) Kill() {
	b.internalChannel <- killBusMessage{}
}
