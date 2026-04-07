package usecase

import (
	"context"
	"fmt"

	"github.com/modern-magic-go/identity"
)

// PasswordLogin 执行用户名密码登录。
func (s *Service) PasswordLogin(ctx context.Context, input PasswordLoginInput) (*LoginResult, error) {
	if input.Realm == "" || input.Identifier == "" || input.Password == "" {
		return nil, fmt.Errorf("realm、identifier、password 不能为空")
	}
	now := s.now()
	normalized := identity.NormalizeIdentifier(input.Identifier)
	ident, err := s.identities.FindByLoginIdentity(ctx, input.Realm, "password", "username", normalized)
	if err != nil {
		return nil, err
	}
	if ident == nil || !ident.IsAvailable() {
		return nil, identity.ErrIdentityUnavailable
	}
	subject, err := s.subjects.GetByID(ctx, ident.SubjectID)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, identity.ErrNilSubject
	}
	if !subject.IsLoginable() {
		return nil, identity.ErrSubjectNotLoginable
	}
	credential, err := s.credentials.FindBySubjectAndRealm(ctx, subject.ID, input.Realm)
	if err != nil {
		return nil, err
	}
	if credential == nil || !credential.IsUsable(now) {
		return nil, identity.ErrCredentialUnavailable
	}
	if err := s.passwordVerifier.VerifyPassword(ctx, input.Password, credential.PasswordHash, credential.PasswordAlgo); err != nil {
		_ = s.credentials.IncrementFailedCount(ctx, credential.ID, now)
		return nil, err
	}
	return &LoginResult{Subject: s.buildSubjectView(subject)}, nil
}
