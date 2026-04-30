package crypto

import (
	"github.com/modern-magic-go/identity"
	"golang.org/x/crypto/bcrypt"
)

// DefaultCost is the default bcrypt cost factor
const DefaultCost = 10

// Bcrypt 实现 CredentialVerifier 接口，用于密码类型凭证的哈希和校验
type Bcrypt struct{}

// Type 返回此 verifier 对应的凭证类型
func (b *Bcrypt) Type() identity.IdentityType {
	return identity.TypePassword
}

// Verify 验证输入的明文密码是否与存储的哈希匹配
func (b *Bcrypt) Verify(storedData, inputData string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(storedData), []byte(inputData))
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Hash 使用 bcrypt 算法对密码进行哈希，返回 cost 可配置
func Hash(password string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify 验证输入的明文密码是否与存储的 bcrypt 哈希匹配
// 匹配返回 nil，不匹配返回 bcrypt.ErrMismatchedHashAndPassword
func Verify(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
