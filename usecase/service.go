package usecase

import (
	"fmt"

	"github.com/modern-magic-go/identity"
)

// Config 定义身份底座用例依赖。
type Config struct {
	Subjects         identity.SubjectRepository
	Identities       identity.IdentityRepository
	Credentials      identity.PasswordCredentialRepository
	PasswordVerifier identity.PasswordVerifier
	Clock            identity.Clock
	IDGenerator      identity.IDGenerator
}

// Service 提供统一用例入口。
type Service struct {
	subjects         identity.SubjectRepository
	identities       identity.IdentityRepository
	credentials      identity.PasswordCredentialRepository
	passwordVerifier identity.PasswordVerifier
	clock            identity.Clock
	idGenerator      identity.IDGenerator
}

// NewService 创建用例服务。
func NewService(cfg Config) (*Service, error) {
	if cfg.Subjects == nil || cfg.Identities == nil || cfg.Credentials == nil {
		return nil, fmt.Errorf("identity repositories are required")
	}
	if cfg.PasswordVerifier == nil {
		return nil, fmt.Errorf("password verifier is required")
	}
	if cfg.Clock == nil || cfg.IDGenerator == nil {
		return nil, fmt.Errorf("clock and id generator are required")
	}
	return &Service{
		subjects:         cfg.Subjects,
		identities:       cfg.Identities,
		credentials:      cfg.Credentials,
		passwordVerifier: cfg.PasswordVerifier,
		clock:            cfg.Clock,
		idGenerator:      cfg.IDGenerator,
	}, nil
}
