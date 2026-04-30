package usecase

import (
	"context"
	"testing"

	"github.com/modern-magic-go/identity"
	"github.com/modern-magic-go/identity/internal/crypto"
	"github.com/modern-magic-go/identity/internal/idgen"
	"github.com/modern-magic-go/identity/internal/store"
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
