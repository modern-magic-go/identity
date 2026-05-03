package core

import (
	"context"

	"github.com/modern-magic-go/identity"
	"github.com/modern-magic-go/identity/internal/crypto"
	"github.com/modern-magic-go/identity/usecase"
)

// IdentityCore 库的公共入口，聚合 store 和 verifier map，对外暴露 5 个方法
type IdentityCore struct {
	store     identity.IdentityStore
	verifiers map[identity.IdentityType]crypto.CredentialVerifier
}

// NewIdentityCore 创建 IdentityCore 实例，内建 PASSWORD (bcrypt) + TOTP verifier
func NewIdentityCore(store identity.IdentityStore) *IdentityCore {
	return &IdentityCore{
		store: store,
		verifiers: map[identity.IdentityType]crypto.CredentialVerifier{
			identity.TypePassword: &crypto.Bcrypt{},
			identity.TypeTOTP:     &crypto.TOTP{},
		},
	}
}

// VerifyCredential 验证调用方提供的标识和凭证是否匹配
func (c *IdentityCore) VerifyCredential(ctx context.Context, input identity.VerifyInput) (identity.VerifyOutput, error) {
	return usecase.VerifyCredential(ctx, c.store, c.verifiers, input)
}

// GetOrInitializeSubjectID 静默解析标识符：有则返回已有 subject_id，无则创建新 subject 并注册凭证
func (c *IdentityCore) GetOrInitializeSubjectID(ctx context.Context, input identity.GetOrInitSubjectInput) (identity.GetOrInitSubjectOutput, error) {
	return usecase.GetOrInitializeSubjectID(ctx, c.store, input)
}

// BindCredential 为已存在的 subject 绑定新凭证
func (c *IdentityCore) BindCredential(ctx context.Context, input identity.BindCredentialInput) error {
	return usecase.BindCredential(ctx, c.store, input)
}

// ListCredentials 列出 subject 在指定 Realm 下所有凭证（不含敏感数据）
func (c *IdentityCore) ListCredentials(ctx context.Context, input identity.ListCredentialsInput) ([]identity.CredentialSummary, error) {
	return usecase.ListCredentials(ctx, c.store, input)
}

