package slave

import (
	"github.com/KIT-MAMID/mamid/msp"
	"os/exec"
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

// Spawn a new Mongod process
//   The Mongod's `--keyfile` is only updated when it is spawned
func (p *ProcessManager) SpawnProcess(m msp.Mongod) (err error) {

	if err = p.createDirSkeleton(m); err != nil {
		return
	}

	if err = p.UpdateKeyfile(m); err != nil {
		return
	}

	args := p.buildMongodCommandLine(m)
	log.Debugf("spwaning Mongod with arguments: %v", args)
	cmd := exec.Command(p.command, args...)
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		processState, err := cmd.Process.Wait()
		if err != nil {
			log.Errorf("error waiting for process `%v`: %s", cmd, err)
		} else if !processState.Success() {
			log.Errorf("Mongod exited unsuccessfully: %v", processState)
		}
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
