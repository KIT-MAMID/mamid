package slave

import "github.com/KIT-MAMID/mamid/msp"

type MongodConfigurator interface {
	MongodConfiguration(p msp.PortNumber) (msp.Mongod, *msp.Error)
	ApplyMongodConfiguration(m msp.Mongod) *msp.Error
}

type ConcreteMongodConfigurator struct {
}

func (c *ConcreteMongodConfigurator) MongodConfiguration(p msp.PortNumber) (msp.Mongod, *msp.Error) {
	return msp.Mongod{}, nil
}

func (c *ConcreteMongodConfigurator) ApplyMongodConfiguration(m msp.Mongod) *msp.Error {
	return nil
}
