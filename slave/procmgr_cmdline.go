package slave

import (
	"bufio"
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
	"github.com/Masterminds/semver"
	"os/exec"
	"regexp"
)

const mongodMinRequiredVersion = ">= 3.2"

func (p *ProcessManager) buildMongodCommandLine(m msp.Mongod) (args []string) {

	args = []string{
		"--dbpath", p.processDBPathDir(m),
		"--port", fmt.Sprintf("%d", m.Port),
		"--replSet", m.ReplicaSetConfig.ReplicaSetName,
		"--keyFile", p.processKeyfilePath(m),
	}

	switch m.ReplicaSetConfig.ShardingRole {
	case msp.ShardingRoleShardServer:
		args = append(args, "--shardsvr")
	case msp.ShardingRoleConfigServer:
		args = append(args, "--configsvr")
	}

	return args

}

func (p *ProcessManager) checkMongoDVersion() (bool, error) {
	version := ""
	cmd := exec.Command(p.command, "--version")
	stdOut, err := cmd.StdoutPipe()
	defer stdOut.Close()
	if err != nil {
		return false, fmt.Errorf("processmanager.checkMongoDVersion() failed with: %s", err)
	}
	scan := bufio.NewScanner(stdOut)
	if err := cmd.Start(); err != nil {
		return false, fmt.Errorf("processmanager.checkMongoDVersion() failed with: %s", err)
	}
	if scan.Scan() {
		re, _ := regexp.Compile(`v([0-9].*)`)
		res := re.FindStringSubmatch(scan.Text())
		if len(res) >= 2 {
			version = res[1]
		}
	}
	if err := cmd.Wait(); err != nil {
		return false, fmt.Errorf("processmanager.checkMongoDVersion() failed with: %s", err)
	}
	constraint, err := semver.NewConstraint(mongodMinRequiredVersion)
	if err != nil {
		return false, fmt.Errorf("processmanager.checkMongoDVersion() failed with: %s", err)
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return false, fmt.Errorf("processmanager.checkMongoDVersion() failed with: %s", err)
	}

	return constraint.Check(v), nil
}
