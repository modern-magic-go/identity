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

// Subject 用户主体
type Subject struct {
	SubjectID SubjectID
	IsActive  bool
}

// Meta 凭证附属元信息，key/value 由各 IdentityType 约定
type Meta map[string]string

// Credential 原子凭证：记录一个 subject 在某 Realm 下的某种登录方式
type Credential struct {
	SubjectID      SubjectID
	Realm          string
	IdentityType   IdentityType
	Identifier     string
	CredentialData string
	IsActive       bool        // 语义：此登录方式是否被禁（第二道闸）
	SubjectActive  bool        // store 层填充的 Subject 级活跃状态（第一道闸，只读）
	Meta           Meta        // 凭证元信息，存储层 JSON 序列化
}

// CredentialSummary 凭证摘要（脱敏后不含 CredentialData）
type CredentialSummary struct {
	Type       IdentityType
	Identifier string
	IsActive   bool
}
