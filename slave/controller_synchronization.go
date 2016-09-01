package slave

import (
	"github.com/KIT-MAMID/mamid/msp"
	"sync"
)

// thread-safe data structure managing mutexes identified by msp.PortNumber
type busyTable struct {
	mutexes     map[msp.PortNumber]*sync.Mutex
	mutexesLock sync.Mutex
}

func NewBusyTable() *busyTable {
	return &busyTable{
		mutexes:     make(map[msp.PortNumber]*sync.Mutex),
		mutexesLock: sync.Mutex{},
	}
}

func (t *busyTable) AcquireLock(port msp.PortNumber) (mutex *sync.Mutex) {

	t.mutexesLock.Lock()
	// acquire a lock if possible [otherwise there is no process and we need to respawn immediately]
	if _, exists := t.mutexes[port]; !exists {
		t.mutexes[port] = &sync.Mutex{}
	}
	mutex = t.mutexes[port]
	mutex.Lock()
	t.mutexesLock.Unlock()

	return mutex

}
