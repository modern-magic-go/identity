package crypto

import "github.com/modern-magic-go/identity"

// CredentialVerifier 凭证校验策略，按 IdentityType 注册
type CredentialVerifier interface {
	Type() identity.IdentityType
	Verify(storedData, inputData string) (bool, error)
}
