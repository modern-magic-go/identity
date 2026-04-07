package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/modern-magic-go/identity"
)

// SendVerifyChallenge 发起验证码挑战。
func (s *Service) SendVerifyChallenge(ctx context.Context, input SendVerifyChallengeInput) (*SendVerifyChallengeResult, error) {
	if input.Realm == "" || input.Mobile == "" {
		return nil, fmt.Errorf("realm、mobile 不能为空")
	}
	if input.BizType == "" {
		input.BizType = "login"
	}
	now := s.now()

	// Generate verification code
	code, err := s.verifyCodeGenerator.Generate(ctx, s.verifyCodeLength)
	if err != nil {
		return nil, fmt.Errorf("生成验证码失败: %w", err)
	}

	// Send verification code via SMS
	sentAt, err := s.verifyCodeSender.Send(ctx, input.Mobile, code, input.BizType)
	if err != nil {
		return nil, fmt.Errorf("发送验证码失败: %w", err)
	}

	// Generate challenge ID
	challengeID, err := s.idGenerator.NewID("chg")
	if err != nil {
		return nil, fmt.Errorf("生成挑战ID失败: %w", err)
	}

	normalized := s.core.NormalizeIdentifier(input.Mobile)
	challenge, findErr := s.challenges.FindByIdentity(ctx, input.Realm, "sms", "mobile", normalized, input.BizType)
	if findErr != nil {
		challenge = &identity.VerifyChallenge{}
	}
	challenge.ChallengeID = challengeID
	challenge.Realm = input.Realm
	challenge.Provider = "sms"
	challenge.IdentityType = "mobile"
	challenge.Identifier = normalized
	challenge.BizType = input.BizType
	challenge.VerifyCode = code
	challenge.MaxAttempt = 5
	challenge.UsedCount = 0
	challenge.ExpireAt = now.Add(5 * time.Minute)
	challenge.Status = identity.ChallengeStatusPending
	challenge.CreatedAt = now
	if err := s.challenges.Save(ctx, challenge); err != nil {
		return nil, err
	}
	return &SendVerifyChallengeResult{
		ChallengeID: challengeID,
		Phone:       input.Mobile,
		Scene:       input.BizType,
		VerifyCode:  code,
		ExpireAt:    challenge.ExpireAt,
		ExpireIn:    maxDurationSeconds(challenge.ExpireAt.Sub(now)),
		SentAt:      sentAt,
	}, nil
}

// VerifyChallengeLogin 执行验证码登录。
func (s *Service) VerifyChallengeLogin(ctx context.Context, input VerifyChallengeLoginInput) (*LoginResult, error) {
	if input.Realm == "" || input.Mobile == "" || input.VerifyCode == "" {
		return nil, fmt.Errorf("realm、mobile、verify_code 不能为空")
	}
	now := s.now()
	challenge, err := s.challenges.FindByChallengeID(ctx, input.ChallengeID)
	if err != nil {
		_ = s.createAudit(ctx, s.buildAudit(nil, input.Realm, "sms", "sms", "mobile", input.Mobile, "challenge_not_found", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}
	if err := s.core.EnsureVerifyChallengeAvailable(challenge, now); err != nil {
		_ = s.challenges.IncrementUsedCount(ctx, challenge.ChallengeID, now)
		_ = s.createAudit(ctx, s.buildAudit(nil, input.Realm, "sms", "sms", "mobile", input.Mobile, "challenge_invalid", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}
	if challenge.VerifyCode != input.VerifyCode {
		_ = s.challenges.IncrementUsedCount(ctx, challenge.ChallengeID, now)
		_ = s.createAudit(ctx, s.buildAudit(nil, input.Realm, "sms", "sms", "mobile", input.Mobile, "verify_code_invalid", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, fmt.Errorf("验证码不正确")
	}
	ident, err := s.identities.FindByLoginIdentity(ctx, input.Realm, "sms", "mobile", s.core.NormalizeIdentifier(input.Mobile))
	if err != nil {
		_ = s.challenges.TouchUsed(ctx, challenge.ChallengeID, now)
		_ = s.createAudit(ctx, s.buildAudit(nil, input.Realm, "sms", "sms", "mobile", input.Mobile, "identity_not_found", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}
	subject, err := s.subjects.GetByID(ctx, ident.SubjectID)
	if err != nil {
		_ = s.challenges.TouchUsed(ctx, challenge.ChallengeID, now)
		return nil, err
	}
	if err := s.core.EnsureSubjectLoginable(subject); err != nil {
		_ = s.challenges.TouchUsed(ctx, challenge.ChallengeID, now)
		_ = s.createAudit(ctx, s.buildAudit(subjectIDPtr(subject.ID), input.Realm, "sms", "sms", "mobile", input.Mobile, "subject_not_loginable", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 0, now))
		return nil, err
	}
	issued, err := s.sessionManager.Issue(ctx, s.buildSessionIssueRequest(subject.ID, input.Realm, input.Platform, input.DeviceID))
	if err != nil {
		return nil, err
	}
	if err := s.challenges.TouchUsed(ctx, challenge.ChallengeID, now); err != nil {
		_ = s.sessionManager.Revoke(ctx, issued.AccessTokenID, issued.RefreshTokenID)
		return nil, err
	}
	if err := s.identities.TouchLastUsedAt(ctx, ident.ID, now); err != nil {
		_ = s.sessionManager.Revoke(ctx, issued.AccessTokenID, issued.RefreshTokenID)
		return nil, err
	}
	if err := s.createSessionRef(ctx, &identity.SessionRef{
		SubjectID: subject.ID,
		Realm:     input.Realm,
		TokenID:   issued.AccessTokenID,
		Platform:  defaultString(input.Platform, "app"),
		DeviceID:  input.DeviceID,
		IssuedAt:  now,
		ExpireAt:  issued.AccessExpiry,
		Status:    identity.SessionRefStatusActive,
		CreatedAt: now,
	}); err != nil {
		_ = s.sessionManager.Revoke(ctx, issued.AccessTokenID, issued.RefreshTokenID)
		return nil, err
	}
	if err := s.createAudit(ctx, s.buildAudit(subjectIDPtr(subject.ID), input.Realm, "sms", "sms", "mobile", input.Mobile, "", input.IP, input.UserAgent, input.DeviceInfo, input.TraceID, 1, now)); err != nil {
		_ = s.sessionManager.Revoke(ctx, issued.AccessTokenID, issued.RefreshTokenID)
		return nil, err
	}
	return &LoginResult{Subject: s.buildSubjectView(subject), Session: s.buildSessionPair(issued, now)}, nil
}
