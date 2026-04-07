package identity

import (
	"context"
	"time"
)

// SubjectRepository 定义主体仓储契约。
type SubjectRepository interface {
	GetByID(ctx context.Context, id int64) (*Subject, error)
	GetBySubjectNo(ctx context.Context, subjectNo string) (*Subject, error)
	Save(ctx context.Context, subject *Subject) error
}

// IdentityRepository 定义身份仓储契约。
type IdentityRepository interface {
	FindByLoginIdentity(ctx context.Context, realm, provider, identityType, identifier string) (*Identity, error)
	FindBySubjectID(ctx context.Context, subjectID int64) ([]*Identity, error)
	Save(ctx context.Context, identity *Identity) error
}

// PasswordCredentialRepository 定义密码凭证仓储契约。
type PasswordCredentialRepository interface {
	FindBySubjectAndRealm(ctx context.Context, subjectID int64, realm string) (*PasswordCredential, error)
	Save(ctx context.Context, credential *PasswordCredential) error
	IncrementFailedCount(ctx context.Context, credentialID int64, now time.Time) error
}

// TotpCredentialRepository 定义 TOTP 凭证仓储契约。
type TotpCredentialRepository interface {
	FindBySubjectAndRealm(ctx context.Context, subjectID int64, realm string) (*TotpCredential, error)
}

// PasswordVerifier 定义密码校验器契约。
type PasswordVerifier interface {
	VerifyPassword(ctx context.Context, plainPassword, storedHash, algorithm string) error
}

// Clock 定义时间源契约。
type Clock interface {
	Now() time.Time
}

// IDGenerator 定义标识生成契约。
type IDGenerator interface {
	NewID(prefix string) (string, error)
}
