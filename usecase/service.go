package usecase

import (
	"fmt"

	"github.com/modern-magic-go/identity"
	"github.com/modern-magic-go/identity/internal/core"
)

// Config 定义身份底座用例依赖。
type Config struct {
	Subjects            identity.SubjectRepository
	Identities          identity.IdentityRepository
	Credentials         identity.PasswordCredentialRepository
	TotpCredentials     identity.TotpCredentialRepository
	Challenges          identity.VerifyChallengeRepository
	Audits              identity.LoginAuditRepository
	SessionRefs         identity.SessionRefRepository
	PasswordVerifier    identity.PasswordVerifier
	VerifyCodeGenerator identity.VerifyCodeGenerator
	VerifyCodeSender    identity.VerifyCodeSender
	VerifyCodeVerifier  identity.VerifyCodeVerifier
	VerifyCodeLength    int
	SessionManager      identity.SessionManager
	Clock               identity.Clock
	IDGenerator         identity.IDGenerator
}

// Service 提供统一用例入口。
type Service struct {
	core                *core.Service
	subjects            identity.SubjectRepository
	identities          identity.IdentityRepository
	credentials         identity.PasswordCredentialRepository
	totpCredentials     identity.TotpCredentialRepository
	challenges          identity.VerifyChallengeRepository
	audits              identity.LoginAuditRepository
	sessionRefs         identity.SessionRefRepository
	passwordVerifier    identity.PasswordVerifier
	verifyCodeGenerator identity.VerifyCodeGenerator
	verifyCodeSender    identity.VerifyCodeSender
	verifyCodeVerifier  identity.VerifyCodeVerifier
	verifyCodeLength    int
	sessionManager      identity.SessionManager
	clock               identity.Clock
	idGenerator         identity.IDGenerator
}

// NewService 创建用例服务。
func NewService(cfg Config) (*Service, error) {
	if cfg.Subjects == nil || cfg.Identities == nil || cfg.Credentials == nil || cfg.Challenges == nil || cfg.Audits == nil || cfg.SessionRefs == nil {
		return nil, fmt.Errorf("identity repositories are required")
	}
	if cfg.PasswordVerifier == nil || cfg.VerifyCodeGenerator == nil || cfg.VerifyCodeSender == nil || cfg.VerifyCodeVerifier == nil {
		return nil, fmt.Errorf("identity providers are required")
	}
	if cfg.SessionManager == nil {
		return nil, fmt.Errorf("session manager is required")
	}
	if cfg.Clock == nil || cfg.IDGenerator == nil {
		return nil, fmt.Errorf("clock and id generator are required")
	}
	if cfg.VerifyCodeLength <= 0 {
		cfg.VerifyCodeLength = 6
	}
	return &Service{
		core:                core.NewService(),
		subjects:            cfg.Subjects,
		identities:          cfg.Identities,
		credentials:         cfg.Credentials,
		totpCredentials:     cfg.TotpCredentials,
		challenges:          cfg.Challenges,
		audits:              cfg.Audits,
		sessionRefs:         cfg.SessionRefs,
		passwordVerifier:    cfg.PasswordVerifier,
		verifyCodeGenerator: cfg.VerifyCodeGenerator,
		verifyCodeSender:    cfg.VerifyCodeSender,
		verifyCodeVerifier:  cfg.VerifyCodeVerifier,
		verifyCodeLength:    cfg.VerifyCodeLength,
		sessionManager:      cfg.SessionManager,
		clock:               cfg.Clock,
		idGenerator:         cfg.IDGenerator,
	}, nil
}

// Core 返回规则服务。
func (s *Service) Core() *core.Service {
	return s.core
}
