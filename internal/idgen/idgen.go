package idgen

// IDGenerator 全局唯一 ID 生成器
type IDGenerator interface {
	Generate() int64
}
