package identity

import "strconv"

// SubjectID 全局唯一用户标识，内部存储为 string，兼容 Snowflake int64 和 UUID
type SubjectID string

// SubjectIDFromInt64 从 Snowflake int64 构造 SubjectID
func SubjectIDFromInt64(id int64) SubjectID {
	return SubjectID(strconv.FormatInt(id, 10))
}

// SubjectIDFromString 从 string（如 UUID）构造 SubjectID
func SubjectIDFromString(id string) SubjectID {
	return SubjectID(id)
}

// Realm 领域/命名空间，账号池的物理隔离单位
type Realm = string

// IdentityType 凭证类型
type IdentityType string

const (
	TypePassword      IdentityType = "PASSWORD"
	TypeWechatOpenID  IdentityType = "WECHAT_OPENID"
	TypeWechatUnionID IdentityType = "WECHAT_UNIONID"
	TypeEmail         IdentityType = "EMAIL"
	TypeTOTP          IdentityType = "TOTP"
	TypeSMS           IdentityType = "SMS"
)

// Credential 原子凭证：记录一个 subject 在某 Realm 下的某种登录方式
type Credential struct {
	SubjectID      SubjectID
	Realm          string
	IdentityType   IdentityType
	Identifier     string
	CredentialData string
	IsActive       bool
}

// CredentialSummary 凭证摘要（脱敏后不含 CredentialData）
type CredentialSummary struct {
	Type       IdentityType
	Identifier string
	IsActive   bool
}
