package main

import (
	"reflect"
)

type Bus struct {
	channels []chan interface{}
}

func (b *Bus) GetNewChannel() chan interface{} {
	channel := make(chan interface{})
	b.channels = append(b.channels, channel)
	//TODO Interrupt Bus.Run Select when adding a new channel
	return channel
}

func (b *Bus) Run() {
	for {
		selects := make([]reflect.SelectCase, len(b.channels))
		for i, channel := range b.channels {
			selects[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(channel)}
		}

		chosen, recv, recvOk := reflect.Select(selects)
		if recvOk {
			for i, channel := range b.channels {
				if i != chosen {
					channel<-recv
				}
			}
		}
	}
}
