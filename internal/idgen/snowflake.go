package idgen

import "github.com/bwmarrin/snowflake"

// Snowflake 基于 Twitter Snowflake 算法的全局 ID 生成器
type Snowflake struct {
	node *snowflake.Node
}

// New 创建 Snowflake 实例，nodeID 范围 0-1023
func New(nodeID int64) (*Snowflake, error) {
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		return nil, err
	}
	return &Snowflake{node: node}, nil
}

// Generate 生成全局唯一 int64 ID
func (s *Snowflake) Generate() int64 {
	return s.node.Generate().Int64()
}
