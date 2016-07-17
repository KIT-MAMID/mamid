package slave

import "github.com/KIT-MAMID/mamid/msp"

type ProcessManager struct {
	RootDataDirectory string
}

func (p *ProcessManager) spawnProcess(m msp.Mongod) error {
	return nil
}

func (p *ProcessManager) runningProcesses(m msp.Mongod) []msp.PortNumber {
	return nil
}

func (p *ProcessManager) killProcess(port msp.PortNumber) error {
	return nil
}

func (p *ProcessManager) killProcesses() error {
	return nil
}
