package masterapi

import (
	"fmt"
)

func (s *Slave) assertNoZeroFieldsSet() error {
	return nil
}

func concatErrors(err1, err2 error) error {
	return fmt.Errorf("%s: %s", err1.Error(), err2.Error())
}

func assertIsPortNumber(u uint) error {
	// TODO nicer way to get number of bits in PortNumber?
	if u < 1 || u > 2^16-1 {
		return fmt.Errorf("a port number must be in [%d, %d]", 1, 2^16-1)
	}
	return nil
}
