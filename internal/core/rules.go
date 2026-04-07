package core

import (
	"fmt"
	"github.com/modern-magic-go/identity"
	"time"
)

// Service 提供身份底座通用规则。
type Service struct{}

// NewService 创建规则服务。
func NewService() *Service {
	return &Service{}
}

// EnsureSubjectLoginable 校验主体是否允许登录。
func (s *Service) EnsureSubjectLoginable(subject *identity.Subject) error {
	if subject == nil {
		return identity.ErrNilSubject
	}
	switch subject.Status {
	case identity.SubjectStatusActive:
		return nil
	case identity.SubjectStatusFrozen, identity.SubjectStatusPendingActivation:
		return identity.ErrSubjectNotLoginable
	case identity.SubjectStatusDeactivating, identity.SubjectStatusDeactivated:
		return identity.ErrSubjectNotLoginable
	default:
		return fmt.Errorf("subject status %q is not loginable", subject.Status)
	}
}

// NormalizeIdentifier 统一规范化标识。
func (s *Service) NormalizeIdentifier(identifier string) string {
	return identity.NormalizeIdentifier(identifier)
}

// EnsureIdentityAvailable 校验身份是否可用。
func (s *Service) EnsureIdentityAvailable(identity_ *identity.Identity) error {
	if identity_ == nil {
		return identity.ErrIdentityUnavailable
	}
	if !identity_.IsAvailable() {
		return identity.ErrIdentityUnavailable
	}
	return nil
}

// EnsurePasswordCredentialAvailable 校验密码凭证是否可用。
func (s *Service) EnsurePasswordCredentialAvailable(credential *identity.PasswordCredential, now time.Time) error {
	if credential == nil || !credential.IsUsable(now) {
		return identity.ErrCredentialUnavailable
	}
	return nil
}
