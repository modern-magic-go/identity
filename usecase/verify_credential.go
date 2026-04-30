package usecase

import (
	"context"
	"errors"

	"github.com/modern-magic-go/identity"
	"github.com/modern-magic-go/identity/internal/crypto"
)

// VerifyCredential 验证调用方提供的标识和凭证是否匹配
func VerifyCredential(
	ctx context.Context,
	store identity.IdentityStore,
	verifiers map[identity.IdentityType]crypto.CredentialVerifier,
	input identity.VerifyInput,
) (identity.VerifyOutput, error) {
	cred, err := store.FindByRealmTypeIdentifier(ctx, input.Realm, input.IdentityType, input.Identifier)
	if err != nil {
		if errors.Is(err, identity.ErrCredentialNotFound) {
			return identity.VerifyOutput{
				Success:   false,
				ErrorCode: "CREDENTIAL_NOT_FOUND",
				ErrorMsg:  "credential not found for the given realm, type and identifier",
			}, nil
		}
		return identity.VerifyOutput{}, err
	}

	verifier, ok := verifiers[cred.IdentityType]
	if !ok {
		return identity.VerifyOutput{
			Success:   false,
			ErrorCode: "UNSUPPORTED_TYPE",
			ErrorMsg:  "no verifier registered for credential type " + string(cred.IdentityType),
		}, nil
	}

	match, err := verifier.Verify(cred.CredentialData, input.InputData)
	if err != nil {
		return identity.VerifyOutput{}, err
	}
	if !match {
		return identity.VerifyOutput{
			Success:   false,
			ErrorCode: "INVALID_CREDENTIAL",
			ErrorMsg:  "credential verification failed",
		}, nil
	}

	return identity.VerifyOutput{
		Success:   true,
		SubjectID: cred.SubjectID,
	}, nil
}
