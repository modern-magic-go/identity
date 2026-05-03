package usecase

import (
	"context"
	"errors"

	"github.com/modern-magic-go/identity"
)

// GetOrInitializeSubjectID 静默解析标识符：有则返回已有 subject_id，无则创建新 subject 并注册凭证
func GetOrInitializeSubjectID(
	ctx context.Context,
	store identity.IdentityStore,
	input identity.GetOrInitSubjectInput,
) (identity.GetOrInitSubjectOutput, error) {
	cred, err := store.FindByRealmTypeIdentifier(ctx, input.Realm, input.IdentityType, input.Identifier)
	if err == nil {
		if !cred.SubjectActive {
			return identity.GetOrInitSubjectOutput{}, identity.ErrAccountLocked
		}
		return identity.GetOrInitSubjectOutput{
			SubjectID: cred.SubjectID,
			IsNewUser: false,
		}, nil
	}

	if !errors.Is(err, identity.ErrCredentialNotFound) {
		return identity.GetOrInitSubjectOutput{}, err
	}

	subjectID, err := store.CreateSubject(ctx)
	if err != nil {
		return identity.GetOrInitSubjectOutput{}, err
	}

	if err := BindCredential(ctx, store, identity.BindCredentialInput{
		SubjectID:    subjectID,
		Realm:        input.Realm,
		IdentityType: input.IdentityType,
		Identifier:   input.Identifier,
	}); err != nil {
		return identity.GetOrInitSubjectOutput{}, err
	}

	return identity.GetOrInitSubjectOutput{
		SubjectID: subjectID,
		IsNewUser: true,
	}, nil
}
