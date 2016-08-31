package slave

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

func (p *ProcessManager) SpawnProcess(m msp.Mongod) error {
	dbDir := fmt.Sprintf("%s/%s/%d:%s", p.dataDir, DataDBDir, m.Port, m.ReplicaSetConfig.ReplicaSetName)
	if err := unix.Access(dbDir, unix.R_OK|unix.W_OK|unix.X_OK); err != nil {
		if err := unix.Mkdir(dbDir, 0700); err != nil {
			panic(fmt.Sprintf("Could not create a readable and writable directory at %s", dbDir))
		}
	}

	args := []string{"--dbpath", dbDir, "--port", fmt.Sprintf("%d", m.Port), "--replSet", m.ReplicaSetConfig.ReplicaSetName}
	if m.ReplicaSetConfig.ShardingConfigServer {
		args = append(args, "--configsvr")
	}

	cmd := exec.Command(p.command, args...)
	err := cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		cmd.Process.Wait()
		p.killChan <- m.Port
	}()

	p.runningProcesses[m.Port] = cmd
	return nil
}

func (p *ProcessManager) ExistingDataDirectories() (replSetNameByPortNumber map[msp.PortNumber]string, err error) {
	entries, err := ioutil.ReadDir(fmt.Sprintf("%s/%s", p.dataDir, DataDBDir))
	if err != nil {
		return
	}
	replSetNameByPortNumber = make(map[msp.PortNumber]string)
	for _, entry := range entries {
		if entry.IsDir() {
			dirname := entry.Name()
			colonLoc := strings.Index(entry.Name(), ":")
			if colonLoc < 0 {
				log.Errorf("Directory with unparsable name `%s` in db dir", entry.Name())
				continue
			}
			portStr := dirname[0:colonLoc]
			port64, parseErr := strconv.ParseUint(portStr, 10, 16)
			if parseErr != nil {
				log.Errorf("Could not parse port `%s`", portStr)
				continue
			}
			port := msp.PortNumber(port64)
			replSetStr := dirname[colonLoc+1:]
			replSetNameByPortNumber[port] = replSetStr
		} else {
			log.Errorf("File `%s` in db dir. There should be no files in db dir.", entry.Name())
			continue
		}
	}
	return
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

func (p *ProcessManager) DestroyDataDirectory(port msp.PortNumber, replSetName string) error {
	dbDir := fmt.Sprintf("%s/%s/%d:%s", p.dataDir, DataDBDir, port, replSetName)
	return os.RemoveAll(dbDir)
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
