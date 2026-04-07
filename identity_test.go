package identity_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/modern-magic-go/identity"
	"github.com/modern-magic-go/identity/usecase"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fixedIDGenerator struct{ seq int }

func (g *fixedIDGenerator) NewID(prefix string) (string, error) {
	g.seq++
	return fmt.Sprintf("%s%03d", prefix, g.seq), nil
}

type fakePasswordVerifier struct{}

func (fakePasswordVerifier) VerifyPassword(_ context.Context, plainPassword, storedHash, _ string) error {
	if storedHash != "hash:"+plainPassword {
		return errors.New("password mismatch")
	}
	return nil
}

type fakeVerifyCodeGenerator struct{}

func (fakeVerifyCodeGenerator) Generate(_ context.Context, _ int) (string, error) {
	return "123456", nil
}

type fakeVerifyCodeSender struct{}

func (fakeVerifyCodeSender) Send(_ context.Context, _, _, _ string) (time.Time, error) {
	return time.Unix(1700000000, 0).UTC(), nil
}

type fakeVerifyCodeVerifier struct{}

func (fakeVerifyCodeVerifier) Verify(_ context.Context, storedHash, code string) error {
	if storedHash != "code:"+code {
		return errors.New("verify code mismatch")
	}
	return nil
}

func (fakeVerifyCodeVerifier) Hash(_ context.Context, code string) (string, error) {
	return "code:" + code, nil
}

type fakeSubjectRepo struct {
	items map[int64]*identity.Subject
}

func (r *fakeSubjectRepo) GetByID(_ context.Context, id int64) (*identity.Subject, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, errors.New("subject not found")
	}
	return item, nil
}

func (r *fakeSubjectRepo) GetBySubjectNo(_ context.Context, subjectNo string) (*identity.Subject, error) {
	for _, item := range r.items {
		if item.SubjectNo == subjectNo {
			return item, nil
		}
	}
	return nil, errors.New("subject not found")
}

func (r *fakeSubjectRepo) Save(_ context.Context, subject *identity.Subject) error {
	r.items[subject.ID] = subject
	return nil
}

type fakeIdentityRepo struct {
	items map[string]*identity.Identity
}

func (r *fakeIdentityRepo) key(realm, provider, identityType, identifier string) string {
	return realm + "|" + provider + "|" + identityType + "|" + identifier
}

func (r *fakeIdentityRepo) FindByLoginIdentity(_ context.Context, realm, provider, identityType, identifier string) (*identity.Identity, error) {
	item, ok := r.items[r.key(realm, provider, identityType, identifier)]
	if !ok {
		return nil, errors.New("identity not found")
	}
	return item, nil
}

func (r *fakeIdentityRepo) FindBySubjectID(_ context.Context, subjectID int64) ([]*identity.Identity, error) {
	var out []*identity.Identity
	for _, item := range r.items {
		if item.SubjectID == subjectID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *fakeIdentityRepo) Save(_ context.Context, identity *identity.Identity) error {
	r.items[r.key(identity.Realm, identity.Provider, identity.IdentityType, identity.Identifier)] = identity
	return nil
}

func (r *fakeIdentityRepo) TouchLastUsedAt(_ context.Context, identityID int64, now time.Time) error {
	for _, item := range r.items {
		if item.ID == identityID {
			item.LastUsedAt = &now
			return nil
		}
	}
	return errors.New("identity not found")
}

type fakeCredentialRepo struct {
	items map[string]*identity.PasswordCredential
}

func (r *fakeCredentialRepo) key(subjectID int64, realm string) string {
	return fmt.Sprintf("%d|%s", subjectID, realm)
}

func (r *fakeCredentialRepo) FindBySubjectAndRealm(_ context.Context, subjectID int64, realm string) (*identity.PasswordCredential, error) {
	item, ok := r.items[r.key(subjectID, realm)]
	if !ok {
		return nil, errors.New("credential not found")
	}
	return item, nil
}

func (r *fakeCredentialRepo) Save(_ context.Context, credential *identity.PasswordCredential) error {
	r.items[r.key(credential.SubjectID, credential.Realm)] = credential
	return nil
}

func (r *fakeCredentialRepo) IncrementFailedCount(_ context.Context, credentialID int64, now time.Time) error {
	for _, item := range r.items {
		if item.ID == credentialID {
			item.FailedCount++
			item.UpdatedAt = now
			return nil
		}
	}
	return errors.New("credential not found")
}

func (r *fakeCredentialRepo) ResetFailedCount(_ context.Context, credentialID int64, now time.Time) error {
	for _, item := range r.items {
		if item.ID == credentialID {
			item.FailedCount = 0
			item.UpdatedAt = now
			return nil
		}
	}
	return errors.New("credential not found")
}

type fakeChallengeRepo struct {
	mu    sync.Mutex
	items map[string]*identity.VerifyChallenge
}

func (r *fakeChallengeRepo) FindByChallengeID(_ context.Context, challengeID string) (*identity.VerifyChallenge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[challengeID]
	if !ok {
		return nil, errors.New("challenge not found")
	}
	return item, nil
}

func (r *fakeChallengeRepo) FindByIdentity(_ context.Context, realm, provider, identityType, identifier, bizType string) (*identity.VerifyChallenge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.items {
		if item.Realm == realm && item.Provider == provider && item.IdentityType == identityType && item.Identifier == identifier && item.BizType == bizType {
			return item, nil
		}
	}
	return nil, errors.New("challenge not found")
}

func (r *fakeChallengeRepo) Save(_ context.Context, challenge *identity.VerifyChallenge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[challenge.ChallengeID] = challenge
	return nil
}

func (r *fakeChallengeRepo) TouchUsed(_ context.Context, challengeID string, usedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[challengeID]
	if !ok {
		return errors.New("challenge not found")
	}
	item.UsedAt = &usedAt
	item.UsedCount++
	item.Status = identity.ChallengeStatusVerified
	return nil
}

func (r *fakeChallengeRepo) IncrementUsedCount(_ context.Context, challengeID string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[challengeID]
	if !ok {
		return errors.New("challenge not found")
	}
	item.UsedCount++
	if item.UsedCount >= item.MaxAttempt {
		item.Status = identity.ChallengeStatusLocked
	}
	return nil
}

type fakeAuditRepo struct{ records []*identity.LoginAudit }

func (r *fakeAuditRepo) Save(_ context.Context, audit *identity.LoginAudit) error {
	r.records = append(r.records, audit)
	return nil
}

type fakeSessionRefRepo struct{ records []*identity.SessionRef }

func (r *fakeSessionRefRepo) Save(_ context.Context, ref *identity.SessionRef) error {
	r.records = append(r.records, ref)
	return nil
}

func (r *fakeSessionRefRepo) RevokeByTokenID(_ context.Context, tokenID string, revokedAt time.Time) error {
	for _, item := range r.records {
		if item.TokenID == tokenID {
			item.Status = identity.SessionRefStatusRevoked
			item.RevokedAt = &revokedAt
			return nil
		}
	}
	return errors.New("session ref not found")
}

func (r *fakeSessionRefRepo) RevokeBySubjectID(_ context.Context, subjectID int64, revokedAt time.Time) (int64, error) {
	var count int64
	for _, item := range r.records {
		if item.SubjectID == subjectID && item.Status == identity.SessionRefStatusActive {
			item.Status = identity.SessionRefStatusRevoked
			item.RevokedAt = &revokedAt
			count++
		}
	}
	return count, nil
}

type fakeSessionManager struct {
	seq         int
	byAccess    map[string]identity.SessionTokenInfo
	byAccessTok map[string]identity.SessionTokenInfo
	bySubject   map[int64]int64
}

func newFakeSessionManager() *fakeSessionManager {
	return &fakeSessionManager{
		byAccess:    map[string]identity.SessionTokenInfo{},
		byAccessTok: map[string]identity.SessionTokenInfo{},
		bySubject:   map[int64]int64{},
	}
}

func (m *fakeSessionManager) Issue(_ context.Context, req identity.SessionIssueRequest) (identity.SessionIssueResult, error) {
	m.seq++
	accessID := fmt.Sprintf("atk-%d", m.seq)
	refreshID := fmt.Sprintf("rtk-%d", m.seq)
	info := identity.SessionTokenInfo{Subject: fmt.Sprintf("%d", req.SubjectID), AccessTokenID: accessID, RefreshTokenID: refreshID, ExpiresAt: time.Now().Add(time.Hour), Metadata: req.Metadata}
	m.byAccess[accessID] = info
	m.byAccessTok["access-"+accessID] = info
	m.bySubject[req.SubjectID]++
	return identity.SessionIssueResult{AccessToken: "access-" + accessID, RefreshToken: "refresh-" + refreshID, AccessTokenID: accessID, RefreshTokenID: refreshID, AccessExpiry: time.Now().Add(time.Hour), RefreshExpiry: time.Now().Add(2 * time.Hour), Metadata: req.Metadata}, nil
}

func (m *fakeSessionManager) InspectAccessToken(_ context.Context, accessToken string) (identity.SessionTokenInfo, error) {
	info, ok := m.byAccessTok[accessToken]
	if !ok {
		return identity.SessionTokenInfo{}, errors.New("session not found")
	}
	return info, nil
}

func (m *fakeSessionManager) GetAccessSession(_ context.Context, accessTokenID string) (identity.SessionTokenInfo, error) {
	info, ok := m.byAccess[accessTokenID]
	if !ok {
		return identity.SessionTokenInfo{}, errors.New("session not found")
	}
	return info, nil
}

func (m *fakeSessionManager) Revoke(_ context.Context, accessTokenID, _ string) error {
	delete(m.byAccess, accessTokenID)
	return nil
}

func (m *fakeSessionManager) RevokeAllBySubject(_ context.Context, subjectID int64) (int64, error) {
	count := m.bySubject[subjectID]
	m.bySubject[subjectID] = 0
	return count, nil
}

func buildService() (*usecase.Service, *fakeAuditRepo, *fakeSessionRefRepo, *fakeChallengeRepo, *fakeCredentialRepo, *fakeIdentityRepo, *fakeSubjectRepo, *fakeSessionManager) {
	subjects := &fakeSubjectRepo{items: map[int64]*identity.Subject{}}
	identities := &fakeIdentityRepo{items: map[string]*identity.Identity{}}
	credentials := &fakeCredentialRepo{items: map[string]*identity.PasswordCredential{}}
	challenges := &fakeChallengeRepo{items: map[string]*identity.VerifyChallenge{}}
	audits := &fakeAuditRepo{}
	sessionRefs := &fakeSessionRefRepo{}
	sessions := newFakeSessionManager()
	svc, err := usecase.NewService(usecase.Config{
		Subjects:            subjects,
		Identities:          identities,
		Credentials:         credentials,
		Challenges:          challenges,
		Audits:              audits,
		SessionRefs:         sessionRefs,
		PasswordVerifier:    fakePasswordVerifier{},
		VerifyCodeGenerator: fakeVerifyCodeGenerator{},
		VerifyCodeSender:    fakeVerifyCodeSender{},
		VerifyCodeLength:    6,
		VerifyCodeVerifier:  fakeVerifyCodeVerifier{},
		SessionManager:      sessions,
		Clock:               fixedClock{now: time.Unix(1700000000, 0).UTC()},
		IDGenerator:         &fixedIDGenerator{},
	})
	if err != nil {
		panic(err)
	}
	return svc, audits, sessionRefs, challenges, credentials, identities, subjects, sessions
}

func seedPasswordLoginData(subjects *fakeSubjectRepo, identities *fakeIdentityRepo, credentials *fakeCredentialRepo, status identity.SubjectStatus) {
	now := time.Unix(1700000000, 0).UTC()
	subjects.items[1] = &identity.Subject{ID: 1, SubjectNo: "sub-1", SubjectType: "admin", Realm: "admin", Status: status, CreatedAt: now, UpdatedAt: now}
	identities.items["admin|password|username|admin_user"] = &identity.Identity{ID: 10, SubjectID: 1, Realm: "admin", Provider: "password", IdentityType: "username", Identifier: "admin_user", Status: identity.IdentityStatusActive, CreatedAt: now, UpdatedAt: now}
	credentials.items["1|admin"] = &identity.PasswordCredential{ID: 20, SubjectID: 1, Realm: "admin", PasswordHash: "hash:123456", PasswordAlgo: "bcrypt", Status: identity.CredentialStatusActive, CreatedAt: now, UpdatedAt: now}
}

func seedVerifyLoginData(subjects *fakeSubjectRepo, identities *fakeIdentityRepo, challenges *fakeChallengeRepo, status identity.SubjectStatus) {
	now := time.Unix(1700000000, 0).UTC()
	subjects.items[2] = &identity.Subject{ID: 2, SubjectNo: "sub-2", SubjectType: "workbench", Realm: "workbench", Status: status, CreatedAt: now, UpdatedAt: now}
	identities.items["workbench|sms|mobile|13800000001"] = &identity.Identity{ID: 11, SubjectID: 2, Realm: "workbench", Provider: "sms", IdentityType: "mobile", Identifier: "13800000001", Status: identity.IdentityStatusActive, CreatedAt: now, UpdatedAt: now}
	challenges.items["chg-1"] = &identity.VerifyChallenge{ID: 30, ChallengeID: "chg-1", Realm: "workbench", Provider: "sms", IdentityType: "mobile", Identifier: "13800000001", BizType: "login", VerifyCode: "123456", MaxAttempt: 5, UsedCount: 0, ExpireAt: now.Add(5 * time.Minute), Status: identity.ChallengeStatusPending, CreatedAt: now}
}

func TestPasswordLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, audits, sessionRefs, _, credentials, identities, subjects, _ := buildService()
		seedPasswordLoginData(subjects, identities, credentials, identity.SubjectStatusActive)

		result, err := svc.PasswordLogin(context.Background(), usecase.PasswordLoginInput{Realm: "admin", Identifier: "admin_user", Password: "123456", Platform: "web", DeviceID: "dev-1"})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, int64(1), result.Subject.SubjectID)
		require.Equal(t, "admin", result.Subject.Realm)
		require.Equal(t, 1, len(audits.records))
		require.Equal(t, int32(1), audits.records[0].Result)
		require.Equal(t, 1, len(sessionRefs.records))
		require.Equal(t, identity.SessionRefStatusActive, sessionRefs.records[0].Status)
	})

	t.Run("reject blocked subject", func(t *testing.T) {
		svc, audits, _, _, credentials, identities, subjects, _ := buildService()
		seedPasswordLoginData(subjects, identities, credentials, identity.SubjectStatusFrozen)

		result, err := svc.PasswordLogin(context.Background(), usecase.PasswordLoginInput{Realm: "admin", Identifier: "admin_user", Password: "123456"})
		require.Error(t, err)
		require.Nil(t, result)
		require.NotEmpty(t, audits.records)
		require.Equal(t, int32(0), audits.records[0].Result)
	})

	t.Run("reject invalid credential", func(t *testing.T) {
		svc, audits, _, _, credentials, identities, subjects, _ := buildService()
		seedPasswordLoginData(subjects, identities, credentials, identity.SubjectStatusActive)

		result, err := svc.PasswordLogin(context.Background(), usecase.PasswordLoginInput{Realm: "admin", Identifier: "admin_user", Password: "bad"})
		require.Error(t, err)
		require.Nil(t, result)
		require.Equal(t, int32(0), audits.records[0].Result)
	})
}

func TestVerifyChallengeLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, audits, sessionRefs, challenges, _, identities, subjects, _ := buildService()
		seedVerifyLoginData(subjects, identities, challenges, identity.SubjectStatusActive)

		result, err := svc.VerifyChallengeLogin(context.Background(), usecase.VerifyChallengeLoginInput{Realm: "workbench", Mobile: "13800000001", VerifyCode: "123456", ChallengeID: "chg-1", Platform: "app"})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, int64(2), result.Subject.SubjectID)
		require.Equal(t, 1, len(audits.records))
		require.Equal(t, int32(1), audits.records[0].Result)
		require.Equal(t, identity.ChallengeStatusVerified, challenges.items["chg-1"].Status)
		require.Equal(t, 1, len(sessionRefs.records))
	})

	t.Run("reject invalid code", func(t *testing.T) {
		svc, audits, _, challenges, _, identities, subjects, _ := buildService()
		seedVerifyLoginData(subjects, identities, challenges, identity.SubjectStatusActive)

		result, err := svc.VerifyChallengeLogin(context.Background(), usecase.VerifyChallengeLoginInput{Realm: "workbench", Mobile: "13800000001", VerifyCode: "000000", ChallengeID: "chg-1"})
		require.Error(t, err)
		require.Nil(t, result)
		require.Equal(t, int32(0), audits.records[0].Result)
		require.Equal(t, identity.ChallengeStatusPending, challenges.items["chg-1"].Status)
	})
}

func TestLogoutUseCases(t *testing.T) {
	svc, _, sessionRefs, _, credentials, identities, subjects, sessions := buildService()
	seedPasswordLoginData(subjects, identities, credentials, identity.SubjectStatusActive)

	first, err := svc.PasswordLogin(context.Background(), usecase.PasswordLoginInput{Realm: "admin", Identifier: "admin_user", Password: "123456"})
	require.NoError(t, err)
	second, err := svc.PasswordLogin(context.Background(), usecase.PasswordLoginInput{Realm: "admin", Identifier: "admin_user", Password: "123456"})
	require.NoError(t, err)
	_ = second

	logoutCurrent, err := svc.LogoutCurrent(context.Background(), usecase.LogoutCurrentInput{AccessToken: first.Session.AccessToken})
	require.NoError(t, err)
	require.True(t, logoutCurrent.Success)
	require.Equal(t, identity.SessionRefStatusRevoked, sessionRefs.records[0].Status)
	_, ok := sessions.byAccess["atk-1"]
	require.False(t, ok)

	logoutAll, err := svc.LogoutAll(context.Background(), usecase.LogoutAllInput{SubjectID: 1})
	require.NoError(t, err)
	require.True(t, logoutAll.Success)
}
