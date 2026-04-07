package identity_test

import (
	"context"
	"errors"
	"fmt"
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

func (r *fakeIdentityRepo) Save(_ context.Context, ident *identity.Identity) error {
	r.items[r.key(ident.Realm, ident.Provider, ident.IdentityType, ident.Identifier)] = ident
	return nil
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

func buildService() (*usecase.Service, *fakeCredentialRepo, *fakeIdentityRepo, *fakeSubjectRepo) {
	subjects := &fakeSubjectRepo{items: map[int64]*identity.Subject{}}
	identities := &fakeIdentityRepo{items: map[string]*identity.Identity{}}
	credentials := &fakeCredentialRepo{items: map[string]*identity.PasswordCredential{}}
	svc, err := usecase.NewService(usecase.Config{
		Subjects:         subjects,
		Identities:       identities,
		Credentials:      credentials,
		PasswordVerifier: fakePasswordVerifier{},
		Clock:            fixedClock{now: time.Unix(1700000000, 0).UTC()},
		IDGenerator:      &fixedIDGenerator{},
	})
	if err != nil {
		panic(err)
	}
	return svc, credentials, identities, subjects
}

func seedPasswordLoginData(subjects *fakeSubjectRepo, identities *fakeIdentityRepo, credentials *fakeCredentialRepo, status identity.SubjectStatus) {
	now := time.Unix(1700000000, 0).UTC()
	subjects.items[1] = &identity.Subject{ID: 1, SubjectNo: "sub-1", SubjectType: "admin", Realm: "admin", Status: status, CreatedAt: now, UpdatedAt: now}
	identities.items["admin|password|username|admin_user"] = &identity.Identity{ID: 10, SubjectID: 1, Realm: "admin", Provider: "password", IdentityType: "username", Identifier: "admin_user", Status: identity.IdentityStatusActive, CreatedAt: now, UpdatedAt: now}
	credentials.items["1|admin"] = &identity.PasswordCredential{ID: 20, SubjectID: 1, Realm: "admin", PasswordHash: "hash:123456", PasswordAlgo: "bcrypt", Status: identity.CredentialStatusActive, CreatedAt: now, UpdatedAt: now}
}

func TestPasswordLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, credentials, identities, subjects := buildService()
		seedPasswordLoginData(subjects, identities, credentials, identity.SubjectStatusActive)

		result, err := svc.PasswordLogin(context.Background(), usecase.PasswordLoginInput{Realm: "admin", Identifier: "admin_user", Password: "123456", Platform: "web", DeviceID: "dev-1"})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, int64(1), result.Subject.SubjectID)
		require.Equal(t, "admin", result.Subject.Realm)
	})

	t.Run("reject blocked subject", func(t *testing.T) {
		svc, credentials, identities, subjects := buildService()
		seedPasswordLoginData(subjects, identities, credentials, identity.SubjectStatusFrozen)

		result, err := svc.PasswordLogin(context.Background(), usecase.PasswordLoginInput{Realm: "admin", Identifier: "admin_user", Password: "123456"})
		require.ErrorIs(t, err, identity.ErrSubjectNotLoginable)
		require.Nil(t, result)
	})

	t.Run("reject invalid credential", func(t *testing.T) {
		svc, credentials, identities, subjects := buildService()
		seedPasswordLoginData(subjects, identities, credentials, identity.SubjectStatusActive)

		result, err := svc.PasswordLogin(context.Background(), usecase.PasswordLoginInput{Realm: "admin", Identifier: "admin_user", Password: "bad"})
		require.Error(t, err)
		require.Nil(t, result)
		require.Equal(t, 1, credentials.items["1|admin"].FailedCount)
	})
}
