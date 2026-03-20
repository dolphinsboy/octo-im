package store

import "github.com/bwmarrin/snowflake"

type PrimaryKeyAllocator interface {
	NextPrimaryKey() uint64
}

type SnowflakePrimaryKeyAllocator struct {
	node *snowflake.Node
}

var _ PrimaryKeyAllocator = (*SnowflakePrimaryKeyAllocator)(nil)

func NewSnowflakePrimaryKeyAllocator(nodeID uint64) (*SnowflakePrimaryKeyAllocator, error) {
	node, err := snowflake.NewNode(int64(nodeID))
	if err != nil {
		return nil, err
	}
	return &SnowflakePrimaryKeyAllocator{node: node}, nil
}

func (s *SnowflakePrimaryKeyAllocator) NextPrimaryKey() uint64 {
	return uint64(s.node.Generate().Int64())
}
