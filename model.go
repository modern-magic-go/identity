package identity

// SubjectID 全局唯一用户标识，由 Snowflake 算法生成
type SubjectID = int64

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
	SubjectID      int64
	Realm          string
	IdentityType   IdentityType
	Identifier     string
	CredentialData string
}

// CredentialSummary 凭证摘要（脱敏后不含 CredentialData）
type CredentialSummary struct {
	Type       IdentityType
	Identifier string
}
