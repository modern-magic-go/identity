package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/modern-magic-go/identity"
)

// PasswordLogin 执行用户名密码登录。
func (s *Service) PasswordLogin(ctx context.Context, input PasswordLoginInput) (*LoginResult, error) {
	if input.Realm == "" || input.Identifier == "" || input.Password == "" {
		return nil, fmt.Errorf("realm、identifier、password 不能为空")
	}
	now := s.now()
	normalized := s.core.NormalizeIdentifier(input.Identifier)
	ident, err := s.identities.FindByLoginIdentity(ctx, input.Realm, "password", "username", normalized)
	if err != nil {
		_ = s.createAudit(ctx, s.buildAudit(nil, input.Realm, "password", "password", "username", normalized, "identity_not_found", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}
	subject, err := s.subjects.GetByID(ctx, ident.SubjectID)
	if err != nil {
		_ = s.createAudit(ctx, s.buildAudit(subjectIDPtr(ident.SubjectID), input.Realm, "password", "password", "username", normalized, "subject_not_found", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}
	if err := s.core.EnsureSubjectLoginable(subject); err != nil {
		_ = s.createAudit(ctx, s.buildAudit(subjectIDPtr(subject.ID), input.Realm, "password", "password", "username", normalized, "subject_not_loginable", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}
	credential, err := s.credentials.FindBySubjectAndRealm(ctx, subject.ID, input.Realm)
	if err != nil {
		_ = s.createAudit(ctx, s.buildAudit(subjectIDPtr(subject.ID), input.Realm, "password", "password", "username", normalized, "credential_not_found", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}
	if err := s.core.EnsurePasswordCredentialAvailable(credential, now); err != nil {
		_ = s.createAudit(ctx, s.buildAudit(subjectIDPtr(subject.ID), input.Realm, "password", "password", "username", normalized, "credential_unavailable", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}
	if err := s.passwordVerifier.VerifyPassword(ctx, input.Password, credential.PasswordHash, credential.PasswordAlgo); err != nil {
		_ = s.credentials.IncrementFailedCount(ctx, credential.ID, now)
		_ = s.createAudit(ctx, s.buildAudit(subjectIDPtr(subject.ID), input.Realm, "password", "password", "username", normalized, "password_invalid", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}

	// 2FA 仓储在部分部署中可能未装配，缺失时直接跳过二次验证分支，避免登录崩溃。
	if s.totpCredentials != nil {
		totpCred, err := s.totpCredentials.FindBySubjectAndRealm(ctx, subject.ID, input.Realm)
		if err == nil && totpCred != nil && totpCred.Status == identity.CredentialStatusActive {
			twoFAConfig := s.buildTwoFAConfigFromTotpCredential(totpCred)
			return &LoginResult{
				Subject:    s.buildSubjectView(subject),
				Session:    SessionPair{},
				NeedVerify: true,
				TwoFAConfig: &TwoFAConfig{
					Google: &TwoFAGoogleConfig{
						Issuer:  twoFAConfig.Google.Issuer,
						Account: twoFAConfig.Google.Account,
					},
				},
			}, nil
		}
	}

	issued, err := s.sessionManager.Issue(ctx, s.buildSessionIssueRequest(subject.ID, input.Realm, input.Platform, input.DeviceID))
	if err != nil {
		return nil, err
	}
	if err := s.identities.TouchLastUsedAt(ctx, ident.ID, now); err != nil {
		_ = s.sessionManager.Revoke(ctx, issued.AccessTokenID, issued.RefreshTokenID)
		return nil, err
	}
	if err := s.credentials.ResetFailedCount(ctx, credential.ID, now); err != nil {
		_ = s.sessionManager.Revoke(ctx, issued.AccessTokenID, issued.RefreshTokenID)
		return nil, err
	}
	if err := s.createSessionRef(ctx, &identity.SessionRef{
		SubjectID: subject.ID,
		Realm:     input.Realm,
		TokenID:   issued.AccessTokenID,
		Platform:  defaultString(input.Platform, "web"),
		DeviceID:  input.DeviceID,
		IssuedAt:  now,
		ExpireAt:  issued.AccessExpiry,
		Status:    identity.SessionRefStatusActive,
		CreatedAt: now,
	}); err != nil {
		_ = s.sessionManager.Revoke(ctx, issued.AccessTokenID, issued.RefreshTokenID)
		return nil, err
	}
	if err := s.createAudit(ctx, s.buildAudit(subjectIDPtr(subject.ID), input.Realm, "password", "password", "username", normalized, "", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 1, now)); err != nil {
		_ = s.sessionManager.Revoke(ctx, issued.AccessTokenID, issued.RefreshTokenID)
		return nil, err
	}
	return &LoginResult{Subject: s.buildSubjectView(subject), Session: s.buildSessionPair(issued, now)}, nil
}

// EnsurePasswordLoginResult 仅用于避免未使用导入在旧版编译器上的干扰。
var _ = time.Second
