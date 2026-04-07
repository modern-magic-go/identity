package identity

import (
	"context"
	"time"
)

// SubjectRepository 定义主体仓储契约。
type SubjectRepository interface {
	GetByID(ctx context.Context, id int64) (*Subject, error)
	GetBySubjectNo(ctx context.Context, subjectNo string) (*Subject, error)
	Save(ctx context.Context, subject *Subject) error
}

// IdentityRepository 定义身份仓储契约。
type IdentityRepository interface {
	FindByLoginIdentity(ctx context.Context, realm, provider, identityType, identifier string) (*Identity, error)
	FindBySubjectID(ctx context.Context, subjectID int64) ([]*Identity, error)
	Save(ctx context.Context, identity *Identity) error
	TouchLastUsedAt(ctx context.Context, identityID int64, now time.Time) error
}

// PasswordCredentialRepository 定义密码凭证仓储契约。
type PasswordCredentialRepository interface {
	FindBySubjectAndRealm(ctx context.Context, subjectID int64, realm string) (*PasswordCredential, error)
	Save(ctx context.Context, credential *PasswordCredential) error
	IncrementFailedCount(ctx context.Context, credentialID int64, now time.Time) error
	ResetFailedCount(ctx context.Context, credentialID int64, now time.Time) error
}

// TotpCredentialRepository 定义 TOTP 凭证仓储契约。
type TotpCredentialRepository interface {
	FindBySubjectAndRealm(ctx context.Context, subjectID int64, realm string) (*TotpCredential, error)
}

// VerifyChallengeRepository 定义验证码挑战仓储契约。
type VerifyChallengeRepository interface {
	FindByChallengeID(ctx context.Context, challengeID string) (*VerifyChallenge, error)
	FindByIdentity(ctx context.Context, realm, provider, identityType, identifier, bizType string) (*VerifyChallenge, error)
	Save(ctx context.Context, challenge *VerifyChallenge) error
	TouchUsed(ctx context.Context, challengeID string, usedAt time.Time) error
	IncrementUsedCount(ctx context.Context, challengeID string, now time.Time) error
}

// LoginAuditRepository 定义登录审计仓储契约。
type LoginAuditRepository interface {
	Save(ctx context.Context, audit *LoginAudit) error
}

// SessionRefRepository 定义会话索引仓储契约。
type SessionRefRepository interface {
	Save(ctx context.Context, ref *SessionRef) error
	RevokeByTokenID(ctx context.Context, tokenID string, revokedAt time.Time) error
	RevokeBySubjectID(ctx context.Context, subjectID int64, revokedAt time.Time) (int64, error)
}

// PasswordVerifier 定义密码校验器契约。
type PasswordVerifier interface {
	VerifyPassword(ctx context.Context, plainPassword, storedHash, algorithm string) error
}

// VerifyCodeGenerator 定义验证码生成器契约。
type VerifyCodeGenerator interface {
	Generate(ctx context.Context, length int) (string, error)
}

// VerifyCodeSender 定义验证码发送器契约。
type VerifyCodeSender interface {
	Send(ctx context.Context, mobile, code, bizType string) (sentAt time.Time, err error)
}

// VerifyCodeVerifier 定义验证码校验器契约。
type VerifyCodeVerifier interface {
	Verify(ctx context.Context, storedHash, code string) error
	Hash(ctx context.Context, code string) (string, error)
}

// SessionIssueRequest 定义会话签发输入。
type SessionIssueRequest struct {
	SubjectID int64
	Realm     string
	Platform  string
	DeviceID  string
	Metadata  map[string]any
}

// SessionIssueResult 定义会话签发结果。
type SessionIssueResult struct {
	AccessToken    string
	RefreshToken   string
	AccessTokenID  string
	RefreshTokenID string
	AccessExpiry   time.Time
	RefreshExpiry  time.Time
	Metadata       map[string]any
}

// SessionTokenInfo 定义访问令牌关联的会话信息。
type SessionTokenInfo struct {
	Subject        string
	AccessTokenID  string
	RefreshTokenID string
	ExpiresAt      time.Time
	Metadata       map[string]any
}

// SessionManager 定义会话签发与撤销契约。
type SessionManager interface {
	Issue(ctx context.Context, req SessionIssueRequest) (SessionIssueResult, error)
	InspectAccessToken(ctx context.Context, accessToken string) (SessionTokenInfo, error)
	GetAccessSession(ctx context.Context, accessTokenID string) (SessionTokenInfo, error)
	Revoke(ctx context.Context, accessTokenID, refreshTokenID string) error
	RevokeAllBySubject(ctx context.Context, subjectID int64) (int64, error)
}

// Clock 定义时间源契约。
type Clock interface {
	Now() time.Time
}

// IDGenerator 定义标识生成契约。
type IDGenerator interface {
	NewID(prefix string) (string, error)
}
