package core

import (
	"context"
	"testing"
	"time"

	"github.com/modern-magic-go/identity"
	"github.com/modern-magic-go/identity/internal/crypto"
	"github.com/modern-magic-go/identity/internal/store"
	"github.com/pquerna/otp/totp"
)

func setupCore(t *testing.T) *IdentityCore {
	t.Helper()
	return NewIdentityCore(store.NewMockStore())
}

func TestVerifyCredentialSuccess(t *testing.T) {
	mock := store.NewMockStore()
	c := NewIdentityCore(mock)
	ctx := context.Background()

	hash, err := crypto.Hash("secret123", crypto.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	subjectID, err := mock.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = mock.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "bob",
		CredentialData: hash,
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := c.VerifyCredential(ctx, identity.VerifyInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "bob",
		InputData:    "secret123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Success {
		t.Fatal("expected Success=true")
	}
	if out.SubjectID != subjectID {
		t.Fatalf("expected SubjectID=%s, got %s", subjectID, out.SubjectID)
	}
}

func TestVerifyCredentialWrongPassword(t *testing.T) {
	mock := store.NewMockStore()
	c := NewIdentityCore(mock)
	ctx := context.Background()

	hash, _ := crypto.Hash("correct", crypto.DefaultCost)
	subjectID, _ := mock.CreateSubject(ctx)
	mock.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "bob",
		CredentialData: hash,
	})

	out, err := c.VerifyCredential(ctx, identity.VerifyInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "bob",
		InputData:    "wrong",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Success {
		t.Fatal("expected Success=false")
	}
	if out.ErrorCode != "INVALID_CREDENTIAL" {
		t.Fatalf("expected ErrorCode=INVALID_CREDENTIAL, got %s", out.ErrorCode)
	}
}

func TestVerifyCredentialNotFound(t *testing.T) {
	c := setupCore(t)
	ctx := context.Background()

	out, err := c.VerifyCredential(ctx, identity.VerifyInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "nonexistent",
		InputData:    "anything",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Success {
		t.Fatal("expected Success=false")
	}
	if out.ErrorCode != "CREDENTIAL_NOT_FOUND" {
		t.Fatalf("expected ErrorCode=CREDENTIAL_NOT_FOUND, got %s", out.ErrorCode)
	}
}

func TestVerifyCredentialInactive(t *testing.T) {
	mock := store.NewMockStore()
	c := NewIdentityCore(mock)
	ctx := context.Background()

	hash, _ := crypto.Hash("secret123", crypto.DefaultCost)
	subjectID, _ := mock.CreateSubject(ctx)
	mock.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "frozen",
		CredentialData: hash,
	})
	mock.SetInactive(subjectID)

	out, err := c.VerifyCredential(ctx, identity.VerifyInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "frozen",
		InputData:    "secret123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Success {
		t.Fatal("expected Success=false for inactive subject")
	}
	if out.ErrorCode != "ACCOUNT_LOCKED" {
		t.Fatalf("expected ErrorCode=ACCOUNT_LOCKED, got %s", out.ErrorCode)
	}
}

func TestGetOrInitializeSubjectIDNew(t *testing.T) {
	c := setupCore(t)
	ctx := context.Background()

	out, err := c.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "newuser",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.IsNewUser {
		t.Fatal("expected IsNewUser=true")
	}
	if out.SubjectID == "" {
		t.Fatal("expected non-empty SubjectID")
	}
}

func TestGetOrInitializeSubjectIDExisting(t *testing.T) {
	c := setupCore(t)
	ctx := context.Background()

	out1, _ := c.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "existing",
	})
	if !out1.IsNewUser {
		t.Fatal("first call should be new user")
	}

	out2, err := c.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "existing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out2.IsNewUser {
		t.Fatal("second call should not be new user")
	}
	if out2.SubjectID != out1.SubjectID {
		t.Fatalf("expected same SubjectID, got %s and %s", out1.SubjectID, out2.SubjectID)
	}
}

func TestGetOrInitializeSubjectIDInactive(t *testing.T) {
	mock := store.NewMockStore()
	c := NewIdentityCore(mock)
	ctx := context.Background()

	out, _ := c.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "inactive",
	})

	mock.SetInactive(out.SubjectID)

	_, err := c.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "inactive",
	})
	if err != identity.ErrAccountLocked {
		t.Fatalf("expected ErrAccountLocked, got %v", err)
	}
}

func TestBindCredentialSuccess(t *testing.T) {
	c := setupCore(t)
	ctx := context.Background()

	out, _ := c.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "primary",
	})

	err := c.BindCredential(ctx, identity.BindCredentialInput{
		SubjectID:      out.SubjectID,
		Realm:          "users",
		IdentityType:   identity.TypeTOTP,
		Identifier:     "totp_dev",
		CredentialData: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}

	list, err := c.ListCredentials(ctx, identity.ListCredentialsInput{
		SubjectID: out.SubjectID,
		Realm:     "users",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(list))
	}
}

func TestBindCredentialDuplicate(t *testing.T) {
	c := setupCore(t)
	ctx := context.Background()

	out, _ := c.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "admin",
	})

	err := c.BindCredential(ctx, identity.BindCredentialInput{
		SubjectID:      out.SubjectID,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "admin",
		CredentialData: "hash",
	})
	if err != identity.ErrDuplicateCredential {
		t.Fatalf("expected ErrDuplicateCredential, got %v", err)
	}
}

func TestBindCredentialSubjectNotFound(t *testing.T) {
	c := setupCore(t)
	ctx := context.Background()

	err := c.BindCredential(ctx, identity.BindCredentialInput{
		SubjectID:      identity.SubjectIDFromInt64(999),
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "new",
		CredentialData: "hash",
	})
	if err != identity.ErrSubjectNotFound {
		t.Fatalf("expected ErrSubjectNotFound, got %v", err)
	}
}

func TestListCredentialsHasItems(t *testing.T) {
	c := setupCore(t)
	ctx := context.Background()

	out, _ := c.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{
		Realm:        "admins",
		IdentityType: identity.TypePassword,
		Identifier:   "admin",
	})

	list, err := c.ListCredentials(ctx, identity.ListCredentialsInput{
		SubjectID: out.SubjectID,
		Realm:     "admins",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(list))
	}
	if list[0].Type != identity.TypePassword || list[0].Identifier != "admin" {
		t.Fatal("unexpected credential content")
	}
}

func TestListCredentialsEmpty(t *testing.T) {
	mock := store.NewMockStore()
	c := NewIdentityCore(mock)
	ctx := context.Background()

	subjectID, _ := mock.CreateSubject(ctx)

	list, err := c.ListCredentials(ctx, identity.ListCredentialsInput{
		SubjectID: subjectID,
		Realm:     "empty",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

func TestListCredentialsSubjectNotFound(t *testing.T) {
	c := setupCore(t)
	ctx := context.Background()

	list, err := c.ListCredentials(ctx, identity.ListCredentialsInput{
		SubjectID: identity.SubjectIDFromInt64(999),
		Realm:     "users",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list for non-existent subject, got %d", len(list))
	}
}

func TestListCredentialsIncludesIsActive(t *testing.T) {
	mock := store.NewMockStore()
	c := NewIdentityCore(mock)
	ctx := context.Background()

	out, _ := c.GetOrInitializeSubjectID(ctx, identity.GetOrInitSubjectInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "status_test",
	})

	list, err := c.ListCredentials(ctx, identity.ListCredentialsInput{
		SubjectID: out.SubjectID,
		Realm:     "users",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(list))
	}
	if !list[0].IsActive {
		t.Fatal("expected IsActive=true in CredentialSummary")
	}

	// SetInactive 不影响 CredentialSummary.IsActive（仅改变 SubjectActive）
	mock.SetInactive(out.SubjectID)
	list, err = c.ListCredentials(ctx, identity.ListCredentialsInput{
		SubjectID: out.SubjectID,
		Realm:     "users",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !list[0].IsActive {
		t.Fatal("SetInactive 不影响 credential-level IsActive，应仍为 true")
	}
}

func TestTwoFactorAuthEndToEnd(t *testing.T) {
	mock := store.NewMockStore()
	c := NewIdentityCore(mock)
	ctx := context.Background()

	subjectID, err := mock.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := crypto.Hash("admin123", crypto.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	err = mock.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "admins",
		IdentityType:   identity.TypePassword,
		Identifier:     "admin",
		CredentialData: hash,
	})
	if err != nil {
		t.Fatal(err)
	}

	secret, _, err := crypto.GenerateTOTPKey("MyApp", "admin")
	if err != nil {
		t.Fatal(err)
	}
	err = c.BindCredential(ctx, identity.BindCredentialInput{
		SubjectID:      subjectID,
		Realm:          "admins",
		IdentityType:   identity.TypeTOTP,
		Identifier:     "totp_device_1",
		CredentialData: secret,
	})
	if err != nil {
		t.Fatal(err)
	}

	out1, err := c.VerifyCredential(ctx, identity.VerifyInput{
		Realm:        "admins",
		IdentityType: identity.TypePassword,
		Identifier:   "admin",
		InputData:    "admin123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out1.Success {
		t.Fatal("PASSWORD verification failed")
	}
	if out1.SubjectID != subjectID {
		t.Fatalf("PASSWORD SubjectID mismatch: %s vs %s", out1.SubjectID, subjectID)
	}

	list, err := c.ListCredentials(ctx, identity.ListCredentialsInput{
		SubjectID: subjectID,
		Realm:     "admins",
	})
	if err != nil {
		t.Fatal(err)
	}
	hasTOTP := false
	for _, cred := range list {
		if cred.Type == identity.TypeTOTP {
			hasTOTP = true
			break
		}
	}
	if !hasTOTP {
		t.Fatal("expected TOTP in credential list")
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	out2, err := c.VerifyCredential(ctx, identity.VerifyInput{
		Realm:        "admins",
		IdentityType: identity.TypeTOTP,
		Identifier:   "totp_device_1",
		InputData:    code,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out2.Success {
		t.Fatal("TOTP verification failed")
	}
	if out2.SubjectID != subjectID {
		t.Fatalf("TOTP SubjectID mismatch: %s vs %s", out2.SubjectID, subjectID)
	}
}

func TestBindCredentialWithMeta(t *testing.T) {
	mock := store.NewMockStore()
	ctx := context.Background()

	subjectID, _ := mock.CreateSubject(ctx)
	err := mock.BindCredential(ctx, &identity.Credential{
		SubjectID:    subjectID,
		Realm:        "app",
		IdentityType: identity.TypeWechatOpenID,
		Identifier:   "o_xxx",
		Meta:         identity.Meta{"appid": "wx1234567890"},
	})
	if err != nil {
		t.Fatal(err)
	}

	cred, err := mock.FindByRealmTypeIdentifier(ctx, "app", identity.TypeWechatOpenID, "o_xxx")
	if err != nil {
		t.Fatal(err)
	}
	if cred.Meta["appid"] != "wx1234567890" {
		t.Fatalf("expected Meta['appid']='wx1234567890', got %v", cred.Meta)
	}
}

func TestBindCredentialNilMeta(t *testing.T) {
	mock := store.NewMockStore()
	ctx := context.Background()

	subjectID, _ := mock.CreateSubject(ctx)
	err := mock.BindCredential(ctx, &identity.Credential{
		SubjectID:    subjectID,
		Realm:        "app",
		IdentityType: identity.TypeWechatOpenID,
		Identifier:   "o_yyy",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = mock.FindByRealmTypeIdentifier(ctx, "app", identity.TypeWechatOpenID, "o_yyy")
	if err != nil {
		t.Fatal(err)
	}
	// nil 或空 map 均可，不 panic 即通过
}

func TestVerifyCredentialSubjectActiveFalse(t *testing.T) {
	mock := store.NewMockStore()
	c := NewIdentityCore(mock)
	ctx := context.Background()

	hash, _ := crypto.Hash("pw123", crypto.DefaultCost)
	subjectID, _ := mock.CreateSubject(ctx)
	mock.BindCredential(ctx, &identity.Credential{
		SubjectID: subjectID, Realm: "app", IdentityType: identity.TypePassword,
		Identifier: "alice", CredentialData: hash,
	})
	mock.SetInactive(subjectID)

	out, err := c.VerifyCredential(ctx, identity.VerifyInput{
		Realm: "app", IdentityType: identity.TypePassword,
		Identifier: "alice", InputData: "pw123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Success {
		t.Fatal("expected Success=false")
	}
	if out.ErrorCode != "ACCOUNT_LOCKED" {
		t.Fatalf("expected ACCOUNT_LOCKED, got %s", out.ErrorCode)
	}
}
