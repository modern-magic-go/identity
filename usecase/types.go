package usecase

import "time"

// SubjectView 是用例返回的主体摘要。
type SubjectView struct {
	SubjectID   int64
	SubjectNo   string
	SubjectType string
	Realm       string
}

// SessionPair 是用例返回的会话摘要。
type SessionPair struct {
	TokenType        string
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int64
	RefreshExpiresIn int64
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	AccessTokenID    string
	RefreshTokenID   string
}

// LoginResult 是登录成功返回。
type LoginResult struct {
	Subject     SubjectView
	Session     SessionPair
	NeedVerify  bool
	TwoFAConfig *TwoFAConfig
}

// TwoFAConfig 是 2FA 配置。
type TwoFAConfig struct {
	SMS    *TwoFASMSConfig    `json:"sms,omitempty"`
	Google *TwoFAGoogleConfig `json:"google,omitempty"`
}

// TwoFASMSConfig 是短信 2FA 配置。
type TwoFASMSConfig struct {
	Phone string `json:"phone"`
}

// TwoFAGoogleConfig 是 Google 2FA 配置。
type TwoFAGoogleConfig struct {
	Issuer  string `json:"issuer,omitempty"`
	Account string `json:"account,omitempty"`
}

// SendVerifyChallengeResult 是验证码挑战发送结果。
type SendVerifyChallengeResult struct {
	ChallengeID string
	Phone       string
	Scene       string
	VerifyCode  string
	ExpireAt    time.Time
	ExpireIn    int64
	SentAt      time.Time
}

// PasswordLoginInput 定义密码登录输入。
type PasswordLoginInput struct {
	Realm      string
	Identifier string
	Password   string
	Platform   string
	DeviceID   string
	DeviceInfo string
	IP         string
	UserAgent  string
	TraceID    string
}

// SendVerifyChallengeInput 定义验证码发送输入。
type SendVerifyChallengeInput struct {
	Realm      string
	Mobile     string
	BizType    string
	Platform   string
	DeviceInfo string
	IP         string
	UserAgent  string
	TraceID    string
}

// VerifyChallengeLoginInput 定义验证码登录输入。
type VerifyChallengeLoginInput struct {
	Realm       string
	Mobile      string
	VerifyCode  string
	ChallengeID string
	Platform    string
	DeviceID    string
	DeviceInfo  string
	IP          string
	UserAgent   string
	TraceID     string
}

// LogoutCurrentInput 定义当前会话注销输入。
type LogoutCurrentInput struct {
	AccessToken string
}

// LogoutResult 定义注销结果。
type LogoutResult struct {
	Success             bool
	RevokedAccessToken  bool
	RevokedRefreshToken bool
}

// LogoutAllInput 定义全量会话注销输入。
type LogoutAllInput struct {
	SubjectID int64
}

// LogoutAllResult 定义全量会话注销结果。
type LogoutAllResult struct {
	Success         bool
	RevokedSessions int
}
