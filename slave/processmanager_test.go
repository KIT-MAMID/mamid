package slave

import (
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func (p *ProcessManager) GetProcess(port msp.PortNumber) *exec.Cmd {
	return p.runningProcesses[port]
}

var dataDir string = "/tmp/mamid_slave-test-" // Set to a temporary directory during setup.

func TestMain(m *testing.M) {

	var err error
	dataDir, err = ioutil.TempDir(os.TempDir(), "mamid_slave-test-")
	if err != nil {
		log.Errorf("Could not create TempDir for slave test: %s", err)
		return
	}

	log.Infof("Created TempDir for slave test: %s", dataDir)

	processManager := &ProcessManager{dataDir: dataDir}
	processManager.CreateManagedDirs()

	//If the killing does not work this test would run forever otherwise
	go func() {
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}()

	ret := m.Run()

	os.RemoveAll(dataDir)

	os.Exit(ret)
}

func TestProcessManager_SpawnProcess(t *testing.T) {
	var err error

	p := NewProcessManager("./fakemongod.sh", dataDir)
	p.Run()

	err = p.SpawnProcess(msp.Mongod{
		Port: 10,
		ReplicaSetConfig: msp.ReplicaSetConfig{
			ReplicaSetName: "replSet",
		},
	})
	assert.NoError(t, err)

	procs := p.RunningProcesses()
	assert.Equal(t, []msp.PortNumber{10}, procs)

	cmd := p.GetProcess(10)

	if err = cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Error(err)
	}

	assert.Equal(t, []string{
		"./fakemongod.sh",
		"--dbpath",
		filepath.Join(dataDir, "mongods", "10:replSet", "dbpath"),
		"--port",
		"10",
		"--replSet",
		"replSet",
		"--keyFile",
		filepath.Join(dataDir, "mongods", "10:replSet", "conf", "keyfile"),
	}, cmd.Args)

	// cleanup
	err = cmd.Process.Signal(syscall.SIGKILL)
	if err != nil {
		t.Error(err)
	}
}

func TestProcessManager_KillProcess(t *testing.T) {
	var err error

	p := NewProcessManager("./fakemongod.sh", dataDir)
	p.Run()

	err = p.SpawnProcess(msp.Mongod{
		Port: 10,
		ReplicaSetConfig: msp.ReplicaSetConfig{
			ReplicaSetName: "replSet",
		},
	})
	assert.NoError(t, err)

	err = p.SpawnProcess(msp.Mongod{
		Port: 11,
		ReplicaSetConfig: msp.ReplicaSetConfig{
			ReplicaSetName: "replSet",
		},
	})
	assert.NoError(t, err)

	cmd1 := p.GetProcess(10)
	cmd2 := p.GetProcess(11)

	if err = cmd1.Process.Signal(syscall.Signal(0)); err != nil {
		t.Error(err)
	}
	if err = cmd2.Process.Signal(syscall.Signal(0)); err != nil {
		t.Error(err)
	}

	procs := p.RunningProcesses()
	assert.Len(t, procs, 2)
	assert.Contains(t, procs, msp.PortNumber(10))
	assert.Contains(t, procs, msp.PortNumber(10))

	p.KillProcess(10)
	cmd1.Process.Wait() //Less racy tests

	time.Sleep(10 * time.Millisecond) // give goroutines a chance to cleanup
	procs = p.RunningProcesses()
	assert.Equal(t, []msp.PortNumber{11}, procs)

	time.Sleep(10 * time.Millisecond) // give goroutines a chance to cleanup
	if err = cmd1.Process.Signal(syscall.Signal(0)); err == nil {
		t.Error("Process 10 still alive after killing")
	}
	if err = cmd2.Process.Signal(syscall.Signal(0)); err != nil {
		t.Error(err)
	}

	cmd2.Process.Kill()
	cmd2.Process.Wait()               //Less racy tests
	time.Sleep(10 * time.Millisecond) // give goroutines a chance to cleanup

	procs = p.RunningProcesses()
	assert.Empty(t, procs)

	if err = cmd2.Process.Signal(syscall.Signal(0)); err == nil {
		t.Error("Process 11 still alive after killing")
	}

	// cleanup
	cmd1.Process.Signal(syscall.SIGKILL)
	cmd2.Process.Signal(syscall.SIGKILL)
}

func TestProcessManager_KillProcesses(t *testing.T) {
	var err error

	p := NewProcessManager("./fakemongod.sh", dataDir)
	p.Run()

	err = p.SpawnProcess(msp.Mongod{
		Port: 10,
		ReplicaSetConfig: msp.ReplicaSetConfig{
			ReplicaSetName: "replSet",
		},
	})
	if err != nil {
		t.Error(err)
	}

	err = p.SpawnProcess(msp.Mongod{
		Port: 11,
		ReplicaSetConfig: msp.ReplicaSetConfig{
			ReplicaSetName: "replSet",
		},
	})
	if err != nil {
		t.Error(err)
	}

	cmd1 := p.GetProcess(10)
	cmd2 := p.GetProcess(11)

	p.KillProcesses()
	cmd1.Process.Wait() //Less racy tests
	cmd2.Process.Wait() //Less racy tests

	time.Sleep(10 * time.Millisecond) // give goroutines a chance to cleanup

	procs := p.RunningProcesses()
	assert.Empty(t, procs)

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
