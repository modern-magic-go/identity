package identity

import "context"

// IdentityStore 持久化层合约接口，由调用方注入实现
type IdentityStore interface {
	// FindByRealmTypeIdentifier 按 Realm + 类型 + 标识符查找凭证
	FindByRealmTypeIdentifier(ctx context.Context, realm string, identityType IdentityType, identifier string) (*Credential, error)

	// CreateSubject 创建新的用户主体，返回生成的 subject_id
	CreateSubject(ctx context.Context) (SubjectID, error)

	// BindCredential 将凭证绑定到指定 subject
	BindCredential(ctx context.Context, cred *Credential) error

	// ListBySubjectRealm 列出 subject 在指定 Realm 下的所有凭证（不含敏感数据）
	ListBySubjectRealm(ctx context.Context, subjectID SubjectID, realm string) ([]CredentialSummary, error)
}

// TxFunc 是在事务内执行的函数
type TxFunc func(ctx context.Context) error

// TransactionalStore 扩展 IdentityStore，声明支持事务
// 实现者可选满足——不支持事务的实现不需要实现此接口
type TransactionalStore interface {
	IdentityStore

	// WithTransaction 在事务中执行 fn
	// fn 内的 IdentityStore 方法调用共享同一事务
	// fn 返回 nil 提交，否则回滚
	WithTransaction(ctx context.Context, fn TxFunc) error
}
