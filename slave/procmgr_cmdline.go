package slave

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/msp"
)

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
