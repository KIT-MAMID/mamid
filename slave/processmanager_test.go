package slave

import (
	"github.com/KIT-MAMID/mamid/msp"
	"testing"
	"fmt"
	"golang.org/x/sys/unix"
	"os/exec"
	"os"
	"syscall"
	"time"
)

func (p *ProcessManager) GetProcess(port msp.PortNumber) *exec.Cmd {
	return p.runningProcesses[port]
}

const dataDir = "testdir"

func TestMain(m *testing.M) {
	if err := unix.Access(dataDir, unix.R_OK | unix.W_OK | unix.X_OK); err != nil {
		if err := unix.Mkdir(dataDir, 0700); err != nil {
			log.Printf("Could not create a readable and writable directory at %s: %s", dataDir, err)
			return
		}
	}

	dbDir := fmt.Sprintf("%s/%s", dataDir, DataDBDir)
	if err := unix.Access(dbDir, unix.R_OK | unix.W_OK | unix.X_OK); err != nil {
		if err := unix.Mkdir(dbDir, 0700); err != nil {
			log.Printf("Could not create a readable and writable directory at %s: %s", dbDir, err)
			return
		}
	}

	ret := m.Run()

	os.RemoveAll(dataDir)

	os.Exit(ret)
}

func TestProcessManager_SpawnProcess(t *testing.T) {
	var err error

	p := NewProcessManager("sleep 2; echo 1", dataDir)
	p.Run()

	err = p.SpawnProcess(msp.Mongod{
		ReplicaSetName: "replSet",
		Port: 10,
	})
	if err != nil {
		t.Error(err)
	}

	procs := p.RunningProcesses()
	if len(procs) != 1 && procs[0] != 10 {
		t.Errorf("bad running processes list (expected [10]), got %+v", procs)
	}

	cmd := p.GetProcess(10)

	if err = cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Error(err)
	}

	if cmd.Path != "/bin/sh" &&
		cmd.Args[0] != "-c" &&
		cmd.Args[1] != "/usr/bin/env sleep 2; echo 1 --dbpath 'testdir/db/replSet' --port 10 --replSet 'replSet'" {
		t.Error("bad command executed")
	}

	// cleanup
	err = cmd.Process.Signal(syscall.SIGKILL)
	if err != nil {
		t.Error(err)
	}
}

func TestProcessManager_KillProcess(t *testing.T) {
	var err error

	p := NewProcessManager("sleep 2; echo 1", dataDir)
	p.Run()

	err = p.SpawnProcess(msp.Mongod{
		ReplicaSetName: "replSet",
		Port: 10,
	})
	if err != nil {
		t.Error(err)
	}

	err = p.SpawnProcess(msp.Mongod{
		ReplicaSetName: "replSet",
		Port: 11,
	})
	if err != nil {
		t.Error(err)
	}

	cmd1 := p.GetProcess(10)
	cmd2 := p.GetProcess(11)

	if err = cmd1.Process.Signal(syscall.Signal(0)); err != nil {
		t.Error(err)
	}
	if err = cmd2.Process.Signal(syscall.Signal(0)); err != nil {
		t.Error(err)
	}


	procs := p.RunningProcesses()
	if len(procs) != 2 || procs[0] != 10 || procs[1] != 11 {
		t.Errorf("bad running processes list (expected [10, 11]), got %+v", procs)
	}

	p.KillProcess(10)

	procs = p.RunningProcesses()
	if len(procs) != 1 || procs[0] != 11 {
		t.Errorf("bad running processes list (expected [11]), got %+v", procs)
	}

	time.Sleep(2 * time.Millisecond) // give goroutines a chance to cleanup
	if err = cmd1.Process.Signal(syscall.Signal(0)); err == nil {
		t.Error("Process 10 still alive after killing")
	}
	if err = cmd2.Process.Signal(syscall.Signal(0)); err != nil {
		t.Error(err)
	}

	cmd2.Process.Kill()
	time.Sleep(2 * time.Millisecond) // give goroutines a chance to cleanup

	procs = p.RunningProcesses()
	if len(procs) != 0 {
		t.Errorf("bad running processes list (expected []), got %+v", procs)
	}

	if err = cmd2.Process.Signal(syscall.Signal(0)); err == nil {
		t.Error("Process 11 still alive after killing")
	}

	// cleanup
	cmd1.Process.Signal(syscall.SIGKILL)
	cmd2.Process.Signal(syscall.SIGKILL)
}

func TestProcessManager_KillProcesses(t *testing.T) {
	var err error

	p := NewProcessManager("sleep 2; echo 1", dataDir)
	p.Run()

	err = p.SpawnProcess(msp.Mongod{
		ReplicaSetName: "replSet",
		Port: 10,
	})
	if err != nil {
		t.Error(err)
	}

	err = p.SpawnProcess(msp.Mongod{
		ReplicaSetName: "replSet",
		Port: 11,
	})
	if err != nil {
		t.Error(err)
	}

	cmd1 := p.GetProcess(10)
	cmd2 := p.GetProcess(11)

	p.KillProcesses()
	time.Sleep(2 * time.Millisecond) // give goroutines a chance to cleanup

	procs := p.RunningProcesses()
	if len(procs) != 0 {
		t.Errorf("bad running processes list (expected []), got %+v", procs)
	}

	if err = cmd1.Process.Signal(syscall.Signal(0)); err == nil {
		t.Error("Process 10 still alive after killing")
	}
	if err = cmd2.Process.Signal(syscall.Signal(0)); err == nil {
		t.Error("Process 11 still alive after killing")
	}

	// cleanup
	cmd1.Process.Signal(syscall.SIGKILL)
	cmd2.Process.Signal(syscall.SIGKILL)
}