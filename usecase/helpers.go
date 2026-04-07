package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/modern-magic-go/identity"
)

func (s *Service) now() time.Time {
	if s.clock == nil {
		return time.Now().UTC()
	}
	return s.clock.Now().UTC()
}

func (s *Service) buildSessionPair(result identity.SessionIssueResult, now time.Time) SessionPair {
	return SessionPair{
		TokenType:        "Bearer",
		AccessToken:      result.AccessToken,
		RefreshToken:     result.RefreshToken,
		ExpiresIn:        maxDurationSeconds(result.AccessExpiry.Sub(now)),
		RefreshExpiresIn: maxDurationSeconds(result.RefreshExpiry.Sub(now)),
		AccessExpiresAt:  result.AccessExpiry,
		RefreshExpiresAt: result.RefreshExpiry,
		AccessTokenID:    result.AccessTokenID,
		RefreshTokenID:   result.RefreshTokenID,
	}
}

func (s *Service) buildSubjectView(subject *identity.Subject) SubjectView {
	if subject == nil {
		return SubjectView{}
	}
	return SubjectView{
		SubjectID:   subject.ID,
		SubjectNo:   subject.SubjectNo,
		SubjectType: subject.SubjectType,
		Realm:       subject.Realm,
	}
}

func (s *Service) createAudit(ctx context.Context, record *identity.LoginAudit) error {
	if s.audits == nil || record == nil {
		return nil
	}
	return s.audits.Save(ctx, record)
}

func (s *Service) createSessionRef(ctx context.Context, ref *identity.SessionRef) error {
	if s.sessionRefs == nil || ref == nil {
		return nil
	}
	return s.sessionRefs.Save(ctx, ref)
}

func (s *Service) buildSessionIssueRequest(subjectID int64, realm, platform, deviceID string) identity.SessionIssueRequest {
	metadata := map[string]any{"realm": realm}
	if strings.TrimSpace(platform) != "" {
		metadata["platform"] = platform
	}
	if strings.TrimSpace(deviceID) != "" {
		metadata["device_id"] = deviceID
	}
	return identity.SessionIssueRequest{
		SubjectID: subjectID,
		Realm:     realm,
		Platform:  platform,
		DeviceID:  deviceID,
		Metadata:  metadata,
	}
}

func (s *Service) buildAudit(subjectID *int64, realm, provider, loginType, identityType, identifier, failReason, ip, userAgent, deviceInfo, traceID string, result int32, loginAt time.Time) *identity.LoginAudit {
	return &identity.LoginAudit{
		SubjectID:        subjectID,
		Realm:            realm,
		Provider:         provider,
		LoginType:        loginType,
		IdentityType:     identityType,
		IdentifierMasked: s.core.MaskIdentifier(identifier),
		Result:           result,
		FailReason:       failReason,
		IP:               ip,
		UserAgent:        userAgent,
		DeviceInfo:       deviceInfo,
		TraceID:          traceID,
		LoginAt:          loginAt,
	}
}

func maxDurationSeconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return int64(duration.Seconds())
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func normalizeProviderAppID(value string) string {
	return strings.TrimSpace(value)
}

func subjectIDPtr(id int64) *int64 {
	return &id
}

func wrapErr(msg string, err error) error {
	if err == nil {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func (s *Service) buildTwoFAConfigFromTotpCredential(totpCred *identity.TotpCredential) *TwoFAConfig {
	if totpCred == nil || totpCred.CredentialMeta == "" {
		return nil
	}
	var meta struct {
		Issuer  string `json:"issuer"`
		Account string `json:"account"`
	}
	if err := json.Unmarshal([]byte(totpCred.CredentialMeta), &meta); err != nil {
		return nil
	}
	return &TwoFAConfig{
		Google: &TwoFAGoogleConfig{
			Issuer:  meta.Issuer,
			Account: meta.Account,
		},
	}
}
