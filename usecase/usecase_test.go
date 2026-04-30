package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/modern-magic-go/identity"
	"github.com/modern-magic-go/identity/internal/crypto"
	"github.com/modern-magic-go/identity/internal/idgen"
	"github.com/modern-magic-go/identity/internal/store"
	"github.com/pquerna/otp/totp"
)

func setupVerifyFixture(t *testing.T) (*store.MockStore, map[identity.IdentityType]crypto.CredentialVerifier) {
	t.Helper()
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	verifiers := map[identity.IdentityType]crypto.CredentialVerifier{
		identity.TypePassword: &crypto.Bcrypt{},
	}
	return mock, verifiers
}

func TestVerifyCredentialSuccess(t *testing.T) {
	store, verifiers := setupVerifyFixture(t)
	ctx := context.Background()

	hash, _ := crypto.Hash("secret123", crypto.DefaultCost)
	id, _ := store.CreateSubject(ctx)
	store.BindCredential(ctx, &identity.Credential{
		SubjectID:      id,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "testuser",
		CredentialData: hash,
	})

	out, err := VerifyCredential(ctx, store, verifiers, identity.VerifyInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "testuser",
		InputData:    "secret123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Success {
		t.Fatal("expected Success=true")
	}
	if out.SubjectID != id {
		t.Fatalf("expected SubjectID=%d, got %d", id, out.SubjectID)
	}
}

func TestVerifyCredentialWrongPassword(t *testing.T) {
	store, verifiers := setupVerifyFixture(t)
	ctx := context.Background()

	hash, _ := crypto.Hash("correct", crypto.DefaultCost)
	id, _ := store.CreateSubject(ctx)
	store.BindCredential(ctx, &identity.Credential{
		SubjectID:      id,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "testuser",
		CredentialData: hash,
	})

	out, err := VerifyCredential(ctx, store, verifiers, identity.VerifyInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "testuser",
		InputData:    "wrong",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Success {
		t.Fatal("expected Success=false for wrong password")
	}
	if out.ErrorCode != "INVALID_CREDENTIAL" {
		t.Fatalf("expected ErrorCode=INVALID_CREDENTIAL, got %s", out.ErrorCode)
	}
}

func TestVerifyCredentialNotFound(t *testing.T) {
	store, verifiers := setupVerifyFixture(t)
	ctx := context.Background()

	out, err := VerifyCredential(ctx, store, verifiers, identity.VerifyInput{
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

func TestGetOrInitSubjectNewUser(t *testing.T) {
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	ctx := context.Background()

	out, err := GetOrInitializeSubjectID(ctx, mock, identity.GetOrInitSubjectInput{
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
	if out.SubjectID <= 0 {
		t.Fatalf("expected positive SubjectID, got %d", out.SubjectID)
	}

	cred, err := mock.FindByRealmTypeIdentifier(ctx, "users", identity.TypePassword, "newuser")
	if err != nil {
		t.Fatal(err)
	}
	if cred.SubjectID != out.SubjectID {
		t.Fatal("credential SubjectID mismatch")
	}
}

func TestGetOrInitSubjectExistingUser(t *testing.T) {
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	ctx := context.Background()

	out1, err := GetOrInitializeSubjectID(ctx, mock, identity.GetOrInitSubjectInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "existing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out1.IsNewUser {
		t.Fatal("first call should be new user")
	}

	out2, err := GetOrInitializeSubjectID(ctx, mock, identity.GetOrInitSubjectInput{
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
		t.Fatalf("expected same SubjectID, got %d and %d", out1.SubjectID, out2.SubjectID)
	}
}

func TestEndToEndCreateAndVerify(t *testing.T) {
	store, verifiers := setupVerifyFixture(t)
	ctx := context.Background()

	subjectID, err := store.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}

	hash, _ := crypto.Hash("alice123", crypto.DefaultCost)
	err = store.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "alice",
		CredentialData: hash,
	})
	if err != nil {
		t.Fatal(err)
	}

	verifyOut, err := VerifyCredential(ctx, store, verifiers, identity.VerifyInput{
		Realm:        "users",
		IdentityType: identity.TypePassword,
		Identifier:   "alice",
		InputData:    "alice123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !verifyOut.Success {
		t.Fatal("expected Success=true")
	}
	if verifyOut.SubjectID != subjectID {
		t.Fatalf("SubjectID mismatch: %d vs %d", verifyOut.SubjectID, subjectID)
	}
}

func setupTOTPFixture(t *testing.T) (*store.MockStore, map[identity.IdentityType]crypto.CredentialVerifier, string) {
	t.Helper()
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	verifiers := map[identity.IdentityType]crypto.CredentialVerifier{
		identity.TypePassword: &crypto.Bcrypt{},
		identity.TypeTOTP:     &crypto.TOTP{},
	}
	secret, _, err := crypto.GenerateTOTPKey("MyApp", "alice")
	if err != nil {
		t.Fatal(err)
	}
	return mock, verifiers, secret
}

func TestVerifyCredentialTOTPSuccess(t *testing.T) {
	store, verifiers, secret := setupTOTPFixture(t)
	ctx := context.Background()

	subjectID, err := store.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = store.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "admins",
		IdentityType:   identity.TypeTOTP,
		Identifier:     "totp_device",
		CredentialData: secret,
	})
	if err != nil {
		t.Fatal(err)
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	out, err := VerifyCredential(ctx, store, verifiers, identity.VerifyInput{
		Realm:        "admins",
		IdentityType: identity.TypeTOTP,
		Identifier:   "totp_device",
		InputData:    code,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Success {
		t.Fatal("expected Success=true for correct TOTP code")
	}
	if out.SubjectID != subjectID {
		t.Fatalf("expected SubjectID=%d, got %d", subjectID, out.SubjectID)
	}
}

func TestVerifyCredentialTOTPWrongCode(t *testing.T) {
	store, verifiers, secret := setupTOTPFixture(t)
	ctx := context.Background()

	subjectID, err := store.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = store.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "admins",
		IdentityType:   identity.TypeTOTP,
		Identifier:     "totp_device",
		CredentialData: secret,
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := VerifyCredential(ctx, store, verifiers, identity.VerifyInput{
		Realm:        "admins",
		IdentityType: identity.TypeTOTP,
		Identifier:   "totp_device",
		InputData:    "000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Success {
		t.Fatal("expected Success=false for wrong TOTP code")
	}
	if out.ErrorCode != "INVALID_CREDENTIAL" {
		t.Fatalf("expected ErrorCode=INVALID_CREDENTIAL, got %s", out.ErrorCode)
	}
}

func TestVerifyCredentialTOTPNoVerifier(t *testing.T) {
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	verifiers := map[identity.IdentityType]crypto.CredentialVerifier{
		identity.TypePassword: &crypto.Bcrypt{},
	}
	ctx := context.Background()

	subjectID, err := mock.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}
	secret, _, _ := crypto.GenerateTOTPKey("MyApp", "alice")
	mock.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "admins",
		IdentityType:   identity.TypeTOTP,
		Identifier:     "totp_device",
		CredentialData: secret,
	})

	out, err := VerifyCredential(ctx, mock, verifiers, identity.VerifyInput{
		Realm:        "admins",
		IdentityType: identity.TypeTOTP,
		Identifier:   "totp_device",
		InputData:    "123456",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Success {
		t.Fatal("expected Success=false when TOTP verifier not registered")
	}
	if out.ErrorCode != "UNSUPPORTED_TYPE" {
		t.Fatalf("expected ErrorCode=UNSUPPORTED_TYPE, got %s", out.ErrorCode)
	}
}

func TestVerifyCredentialTOTPRealmIsolation(t *testing.T) {
	store, verifiers, secret := setupTOTPFixture(t)
	ctx := context.Background()

	subjectID, err := store.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = store.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "realm_a",
		IdentityType:   identity.TypeTOTP,
		Identifier:     "totp_device",
		CredentialData: secret,
	})
	if err != nil {
		t.Fatal(err)
	}

	code, _ := totp.GenerateCode(secret, time.Now())
	out, err := VerifyCredential(ctx, store, verifiers, identity.VerifyInput{
		Realm:        "realm_b",
		IdentityType: identity.TypeTOTP,
		Identifier:   "totp_device",
		InputData:    code,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Success {
		t.Fatal("expected Success=false for wrong realm")
	}
	if out.ErrorCode != "CREDENTIAL_NOT_FOUND" {
		t.Fatalf("expected ErrorCode=CREDENTIAL_NOT_FOUND, got %s", out.ErrorCode)
	}
}

func TestTwoFactorAuthPasswordAndTOTP(t *testing.T) {
	store, verifiers, secret := setupTOTPFixture(t)
	ctx := context.Background()

	subjectID, err := store.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}

	hash, _ := crypto.Hash("admin123", crypto.DefaultCost)
	err = store.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "admins",
		IdentityType:   identity.TypePassword,
		Identifier:     "admin",
		CredentialData: hash,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = store.BindCredential(ctx, &identity.Credential{
		SubjectID:      subjectID,
		Realm:          "admins",
		IdentityType:   identity.TypeTOTP,
		Identifier:     "totp_device_1",
		CredentialData: secret,
	})
	if err != nil {
		t.Fatal(err)
	}

	out1, err := VerifyCredential(ctx, store, verifiers, identity.VerifyInput{
		Realm:        "admins",
		IdentityType: identity.TypePassword,
		Identifier:   "admin",
		InputData:    "admin123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out1.Success {
		t.Fatal("expected PASSWORD verification to succeed")
	}
	if out1.SubjectID != subjectID {
		t.Fatalf("PASSWORD SubjectID mismatch: %d vs %d", out1.SubjectID, subjectID)
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	out2, err := VerifyCredential(ctx, store, verifiers, identity.VerifyInput{
		Realm:        "admins",
		IdentityType: identity.TypeTOTP,
		Identifier:   "totp_device_1",
		InputData:    code,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out2.Success {
		t.Fatal("expected TOTP verification to succeed")
	}
	if out2.SubjectID != subjectID {
		t.Fatalf("TOTP SubjectID mismatch: %d vs %d", out2.SubjectID, subjectID)
	}
}

func TestBindCredentialSuccess(t *testing.T) {
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	ctx := context.Background()

	subjectID, err := mock.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = BindCredential(ctx, mock, identity.BindCredentialInput{
		SubjectID:      subjectID,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "alice",
		CredentialData: "$2a$10$hashed",
	})
	if err != nil {
		t.Fatal(err)
	}

	found, err := mock.FindByRealmTypeIdentifier(ctx, "users", identity.TypePassword, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if found.SubjectID != subjectID {
		t.Fatalf("expected SubjectID=%d, got %d", subjectID, found.SubjectID)
	}
}

func TestBindCredentialDuplicate(t *testing.T) {
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	ctx := context.Background()

	subjectID, _ := mock.CreateSubject(ctx)

	err = BindCredential(ctx, mock, identity.BindCredentialInput{
		SubjectID:      subjectID,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "admin",
		CredentialData: "hash",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = BindCredential(ctx, mock, identity.BindCredentialInput{
		SubjectID:      subjectID,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "admin",
		CredentialData: "hash2",
	})
	if err != identity.ErrDuplicateCredential {
		t.Fatalf("expected ErrDuplicateCredential, got %v", err)
	}
}

func TestBindCredentialSubjectNotFound(t *testing.T) {
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	ctx := context.Background()

	err = BindCredential(ctx, mock, identity.BindCredentialInput{
		SubjectID:      999,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "new",
		CredentialData: "hash",
	})
	if err != identity.ErrSubjectNotFound {
		t.Fatalf("expected ErrSubjectNotFound, got %v", err)
	}
}

func TestListCredentialsHasCredentials(t *testing.T) {
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	ctx := context.Background()

	subjectID, _ := mock.CreateSubject(ctx)
	mock.BindCredential(ctx, &identity.Credential{
		SubjectID: subjectID, Realm: "admins", IdentityType: identity.TypePassword, Identifier: "admin", CredentialData: "hash",
	})
	mock.BindCredential(ctx, &identity.Credential{
		SubjectID: subjectID, Realm: "admins", IdentityType: identity.TypeTOTP, Identifier: "totp_dev", CredentialData: "secret",
	})

	list, err := ListCredentials(ctx, mock, identity.ListCredentialsInput{
		SubjectID: subjectID,
		Realm:     "admins",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(list))
	}
	found := make(map[identity.IdentityType]string)
	for _, c := range list {
		found[c.Type] = c.Identifier
	}
	if found[identity.TypePassword] != "admin" || found[identity.TypeTOTP] != "totp_dev" {
		t.Fatalf("unexpected credential content: %v", found)
	}
}

func TestListCredentialsEmpty(t *testing.T) {
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	ctx := context.Background()

	subjectID, _ := mock.CreateSubject(ctx)

	list, err := ListCredentials(ctx, mock, identity.ListCredentialsInput{
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
	gen, err := idgen.New(1)
	if err != nil {
		t.Fatal(err)
	}
	mock := store.NewMockStore(gen)
	ctx := context.Background()

	list, err := ListCredentials(ctx, mock, identity.ListCredentialsInput{
		SubjectID: 999,
		Realm:     "users",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list for non-existent subject, got %d", len(list))
	}
}
