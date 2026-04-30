package crypto

import (
	"github.com/modern-magic-go/identity"
	"github.com/pquerna/otp/totp"
)

// TOTP 实现 CredentialVerifier 接口，用于 TOTP 动态码校验
type TOTP struct{}

// Type 返回此 verifier 对应的凭证类型
func (t *TOTP) Type() identity.IdentityType {
	return identity.TypeTOTP
}

// Verify 验证 TOTP 码是否与存储的 secret 匹配
// storedData = base32 编码的 TOTP secret（即 CredentialData）
// inputData  = 用户输入的 6 位数字码
func (t *TOTP) Verify(storedData, inputData string) (bool, error) {
	valid := totp.Validate(inputData, storedData)
	return valid, nil
}

// GenerateTOTPKey 生成 TOTP 密钥对，返回 secret（存储到 CredentialData）和 otpauth:// URL（用于生成 QR 码）
func GenerateTOTPKey(issuer, accountName string) (secret string, url string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

var _ CredentialVerifier = (*TOTP)(nil)
