package slave

import "github.com/KIT-MAMID/mamid/msp"

type MongodConfigurator interface {
	MongodConfiguration(p msp.PortNumber) (msp.Mongod, *msp.SlaveError)
	ApplyMongodConfiguration(m msp.Mongod) *msp.SlaveError
}

type ConcreteMongodConfigurator struct {
}

func (c *ConcreteMongodConfigurator) MongodConfiguration(p msp.PortNumber) (msp.Mongod, *msp.SlaveError) {
	return msp.Mongod{}, nil
}

func (c *ConcreteMongodConfigurator) ApplyMongodConfiguration(m msp.Mongod) *msp.SlaveError {
	return nil
}
