package master

import (
	"github.com/jinzhu/gorm"
)

func (c *ClusterAllocator) pqRiskGroups(tx *gorm.DB) map[interface{}]interface{} {
	return make(map[interface{}]interface{})
}

func (c *ClusterAllocator) pqReplicaSets(tx *gorm.DB) interface{} {
	return nil
}
