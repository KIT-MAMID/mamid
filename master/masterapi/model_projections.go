package masterapi

import (
	"fmt"
	"github.com/KIT-MAMID/mamid/model"
)

func (s *Slave) assertNoZeroFieldsSet() error {
	return nil
}

func concatErrors(err1, err2 error) error {
	return fmt.Errorf("%s: %s", err1.Error(), err2.Error())
}

func assertIsPortNumber(u uint) error {
	if u < uint(model.PortNumberMin) || u > uint(model.PortNumberMax) {
		return fmt.Errorf("a port number must be in [%d, %d]", 1, 2^16-1)
	}
	return nil
}
