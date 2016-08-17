package slave

import (
	"github.com/KIT-MAMID/mamid/msp"
	"os/exec"
	"fmt"
	"strings"
	"golang.org/x/sys/unix"
)

type ProcessManager struct {
	command string
	dataDir string
	runningProcesses map[msp.PortNumber]*exec.Cmd
}

func NewProcessManager(command string, dataDir string) ProcessManager {
	return ProcessManager{
		command: command,
		dataDir: dataDir,
		runningProcesses: make(map[msp.PortNumber]*exec.Cmd),
	};
}

func (p *ProcessManager) SpawnProcess(m msp.Mongod) error {

	dbDir := fmt.Sprintf("%s/%s/%s", p.dataDir, DataDBDir, m.ReplicaSetName)
	if err := unix.Access(dbDir, unix.R_OK | unix.W_OK | unix.X_OK); err != nil {
		if err := unix.Mkdir(dbDir, 0700); err != nil {
			panic("Could not create a readable and writable directory at %s")
		}
	}

	escName := strings.Replace(m.ReplicaSetName, "'", "'\\''", -1)
	sh := fmt.Sprintf("/usr/bin/env %s --dbpath '%s/%s/%s' --port %d --replSet '%s'", p.command, p.dataDir, DataDBDir, escName, m.Port, escName)
	cmd := exec.Command("/bin/sh", "-c", sh)
	err := cmd.Start()

	if err != nil {
		return err
	}
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
		delete(p.runningProcesses, port)
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
