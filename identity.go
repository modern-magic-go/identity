package identity

import (
	"strings"
	"time"
)

// SubjectStatus 表示主体状态。
type SubjectStatus string

const (
	SubjectStatusPendingActivation SubjectStatus = "pending_activation"
	SubjectStatusActive            SubjectStatus = "active"
	SubjectStatusFrozen            SubjectStatus = "frozen"
	SubjectStatusDeactivating      SubjectStatus = "deactivating"
	SubjectStatusDeactivated       SubjectStatus = "deactivated"
)

// IdentityStatus 表示身份绑定状态。
type IdentityStatus string

const (
	IdentityStatusActive  IdentityStatus = "active"
	IdentityStatusUnbound IdentityStatus = "unbound"
	IdentityStatusBlocked IdentityStatus = "blocked"
)

// CredentialStatus 表示凭证状态。
type CredentialStatus string

const (
	CredentialStatusActive  CredentialStatus = "active"
	CredentialStatusBlocked CredentialStatus = "blocked"
	CredentialStatusRevoked CredentialStatus = "revoked"
)

// ChallengeStatus 表示验证码挑战状态。
type ChallengeStatus string

const (
	ChallengeStatusPending  ChallengeStatus = "pending"
	ChallengeStatusVerified ChallengeStatus = "verified"
	ChallengeStatusExpired  ChallengeStatus = "expired"
	ChallengeStatusLocked   ChallengeStatus = "locked"
)

// SessionRefStatus 表示会话索引状态。
type SessionRefStatus string

const (
	SessionRefStatusActive  SessionRefStatus = "active"
	SessionRefStatusRevoked SessionRefStatus = "revoked"
	SessionRefStatusExpired SessionRefStatus = "expired"
)

// Subject 表示身份底座主体。
type Subject struct {
	ID          int64
	SubjectNo   string
	SubjectType string
	Realm       string
	Status      SubjectStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// IsLoginable 判断主体当前是否允许登录。
func (s *Subject) IsLoginable() bool {
	if s == nil {
		return false
	}
	switch s.Status {
	case SubjectStatusActive:
		return true
	default:
		return false
	}
}

// Identity 表示身份映射。
type Identity struct {
	ID            int64
	SubjectID     int64
	Realm         string
	Provider      string
	IdentityType  string
	ProviderAppID string
	Identifier    string
	UnionID       string
	Status        IdentityStatus
	LastUsedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NormalizeIdentifier 统一规范化身份标识。
func NormalizeIdentifier(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// IsAvailable 判断身份是否可用。
func (i *Identity) IsAvailable() bool {
	if i == nil {
		return false
	}
	return i.Status == IdentityStatusActive && NormalizeIdentifier(i.Identifier) != ""
}

// PasswordCredential 表示密码凭证。
type PasswordCredential struct {
	ID                int64
	SubjectID         int64
	Realm             string
	PasswordHash      string
	PasswordAlgo      string
	PasswordVersion   int
	NeedReset         bool
	FailedCount       int
	LockedUntil       *time.Time
	PasswordUpdatedAt *time.Time
	Status            CredentialStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TotpCredential 表示 TOTP 凭证。
type TotpCredential struct {
	ID              int64
	SubjectID       int64
	Realm           string
	CredentialValue string
	CredentialMeta  string
	Status          CredentialStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IsUsable 判断密码凭证是否可用。
func (c *PasswordCredential) IsUsable(now time.Time) bool {
	if c == nil {
		return false
	}
	if c.Status != CredentialStatusActive {
		return false
	}
	if c.LockedUntil != nil && c.LockedUntil.After(now) {
		return false
	}
	return strings.TrimSpace(c.PasswordHash) != ""
}

// VerifyChallenge 表示验证码挑战。
type VerifyChallenge struct {
	ID           int64
	ChallengeID  string
	Realm        string
	Provider     string
	IdentityType string
	Identifier   string
	BizType      string
	VerifyCode   string
	MaxAttempt   int
	UsedCount    int
	ExpireAt     time.Time
	UsedAt       *time.Time
	Status       ChallengeStatus
	CreatedAt    time.Time
}

// IsUsable 判断验证码挑战是否可验证。
func (c *VerifyChallenge) IsUsable(now time.Time) bool {
	if c == nil {
		return false
	}
	if c.Status != ChallengeStatusPending {
		return false
	}
	if !c.ExpireAt.IsZero() && !c.ExpireAt.After(now) {
		return false
	}
	return strings.TrimSpace(c.ChallengeID) != ""
}

// LoginAudit 表示登录审计记录。
type LoginAudit struct {
	ID               int64
	SubjectID        *int64
	Realm            string
	Provider         string
	LoginType        string
	IdentityType     string
	IdentifierMasked string
	Result           int32
	FailReason       string
	IP               string
	UserAgent        string
	DeviceInfo       string
	TraceID          string
	LoginAt          time.Time
}

// SessionRef 表示会话索引。
type SessionRef struct {
	ID        int64
	SubjectID int64
	Realm     string
	TokenID   string
	Platform  string
	DeviceID  string
	IssuedAt  time.Time
	ExpireAt  time.Time
	RevokedAt *time.Time
	Status    SessionRefStatus
	CreatedAt time.Time
}
