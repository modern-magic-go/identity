package identity

import "context"

// IdentityStore 持久化层合约接口，由调用方注入实现
type IdentityStore interface {
	// FindByRealmTypeIdentifier 按 Realm + 类型 + 标识符查找凭证
	FindByRealmTypeIdentifier(ctx context.Context, realm string, identityType IdentityType, identifier string) (*Credential, error)

	// CreateSubject 创建新的用户主体，返回 Snowflake 生成的 subject_id
	CreateSubject(ctx context.Context) (int64, error)

	// BindCredential 将凭证绑定到指定 subject
	BindCredential(ctx context.Context, cred *Credential) error

	// ListBySubjectRealm 列出 subject 在指定 Realm 下的所有凭证（不含敏感数据）
	ListBySubjectRealm(ctx context.Context, subjectID int64, realm string) ([]CredentialSummary, error)
}
