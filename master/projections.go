package master

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
	"github.com/KIT-MAMID/mamid/msp"
)

func mspMongodStateFromExecutionState(s model.MongodExecutionState) (msp.MongodState, error) {
	switch s {
	case model.MongodExecutionStateDestroyed:
		return msp.MongodStateDestroyed, nil
	case model.MongodExecutionStateNotRunning:
		return msp.MongodStateNotRunning, nil
	case model.MongodExecutionStateRecovering:
		return msp.MongodStateRecovering, nil
	case model.MongodExecutionStateRunning:
		return msp.MongodStateRunning, nil
	case model.MongodExecutionStateForceDestroyed:
		return msp.MongodStateForceDestroyed, nil
	default:
		return "", fmt.Errorf("deployer: unable to map `%v` from model.ExecutionState to msp.MongodState", s)
	}
}
