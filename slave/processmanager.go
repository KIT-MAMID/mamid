package slave

import (
	"github.com/KIT-MAMID/mamid/msp"
	"os/exec"
	"time"
)

type ProcessManager struct {
	killChan         chan msp.PortNumber
	command          string
	dataDir          string
	runningProcesses map[msp.PortNumber]*exec.Cmd
}

func NewProcessManager(command string, dataDir string) *ProcessManager {
	return &ProcessManager{
		killChan:         make(chan msp.PortNumber),
		command:          command,
		dataDir:          dataDir,
		runningProcesses: make(map[msp.PortNumber]*exec.Cmd),
	}
}

func (p *ProcessManager) Run() {
	go func() {
		for {
			port := <-p.killChan
			delete(p.runningProcesses, port)
		}
	}()
}

func (p *ProcessManager) HasProcess(port msp.PortNumber) bool {
	_, exists := p.runningProcesses[port]
	return exists
}

func (p *ProcessManager) SpawnProcess(m msp.Mongod) (err error) {

	if err = p.createDirSkeleton(m); err != nil {
		return
	}

	cmd := exec.Command(p.command, p.buildMongodCommandLine(m)...)
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		cmd.Process.Wait()
		p.killChan <- m.Port
	}()

	p.runningProcesses[m.Port] = cmd
	return nil

}

func (p *ProcessManager) RunningProcesses() []msp.PortNumber {
	ports := make([]msp.PortNumber, 0, len(p.runningProcesses))
	for port := range p.runningProcesses {
		ports = append(ports, port)
	}
	return ports
}

func (p *ProcessManager) KillProcess(port msp.PortNumber) error {
	if cmd, exists := p.runningProcesses[port]; exists {
		return cmd.Process.Kill()
	}
	return nil
}

func (p *ProcessManager) KillAfterTimeout(port msp.PortNumber, timeout time.Duration) error {
	if cmd, exists := p.runningProcesses[port]; exists {
		done := make(chan struct{})
		go func() {
			cmd.Process.Wait()
			close(done)
		}()

		select {
		case <-done:
			return nil
		case <-time.After(timeout):
			return cmd.Process.Kill()
		}
	}
	return nil
}

// killProcess is destructive. Even when there was an error (already killed, stuck state, permissions lost), we do not care. The error is purely informational that _something_ went wrong.
// This function is to be used for complete clean restart/shutdown only.
func (p *ProcessManager) KillProcesses() error {
	var err error = nil
	for _, cmd := range p.runningProcesses {
		curErr := cmd.Process.Kill()
		if err == nil {
			err = curErr
		}
	}
	p.runningProcesses = make(map[msp.PortNumber]*exec.Cmd)
	return err
}
