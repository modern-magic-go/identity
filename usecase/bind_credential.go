package usecase

import (
	"context"

	"github.com/modern-magic-go/identity"
)

// BindCredential 为已存在的 subject 绑定新凭证，由 store 层保证同 Realm+Type+Identifier 唯一
func BindCredential(
	ctx context.Context,
	store identity.IdentityStore,
	input identity.BindCredentialInput,
) error {
	cred := &identity.Credential{
		SubjectID:      input.SubjectID,
		Realm:          input.Realm,
		IdentityType:   input.IdentityType,
		Identifier:     input.Identifier,
		CredentialData: input.CredentialData,
	}
	return store.BindCredential(ctx, cred)
}
