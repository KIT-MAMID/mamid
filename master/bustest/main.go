package main

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/master"
	"time"
)

func main() {
	var bus master.Bus
	ch1 := bus.GetNewChannel()
	ch2 := bus.GetNewChannel()
	ch3 := bus.GetNewChannel()

	go func(ch chan interface{}) {
		for i := 1; ; i++ {
			ch <- fmt.Sprintf("Test %d", i)
			fmt.Println(<-ch)
			time.Sleep(1000 * time.Millisecond)
		}
	}(ch1)

	go func(ch chan interface{}) {
		for {
			fmt.Println(<-ch)
			ch <- fmt.Sprintf("Test back")
		}
	}(ch2)

	go func(ch chan interface{}) {
		//When a bus member does not fetch its messages the bus will block
		//select {
		//
		//}
		for {
			fmt.Println(<-ch)
		}
	}(ch3)

	bus.Run()
}
