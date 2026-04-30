package usecase

import (
	"context"

	"github.com/modern-magic-go/identity"
)

// ListCredentials 列出 subject 在指定 Realm 下所有凭证（不含敏感数据），用于判断是否需要 2FA
func ListCredentials(
	ctx context.Context,
	store identity.IdentityStore,
	input identity.ListCredentialsInput,
) ([]identity.CredentialSummary, error) {
	return store.ListBySubjectRealm(ctx, input.SubjectID, input.Realm)
}
