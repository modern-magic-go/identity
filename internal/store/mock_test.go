package store

import (
	"context"
	"testing"

	"github.com/modern-magic-go/identity"
)

func setupMockStore(t *testing.T) *MockStore {
	t.Helper()
	return NewMockStore()
}

func TestMockStoreCreateSubject(t *testing.T) {
	store := setupMockStore(t)
	ctx := context.Background()

	id, err := store.CreateSubject(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("expected non-empty subject ID")
	}

	id2, _ := store.CreateSubject(ctx)
	if id == id2 {
		t.Fatal("expected different subject IDs")
	}
}

func TestMockStoreBindCredential(t *testing.T) {
	store := setupMockStore(t)
	ctx := context.Background()

	id, _ := store.CreateSubject(ctx)

	cred := &identity.Credential{
		SubjectID:      id,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "admin",
		CredentialData: "$2a$10$hashed",
	}

	err := store.BindCredential(ctx, cred)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMockStoreBindDuplicate(t *testing.T) {
	store := setupMockStore(t)
	ctx := context.Background()

	id, _ := store.CreateSubject(ctx)

	cred := &identity.Credential{
		SubjectID:      id,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "admin",
		CredentialData: "hash",
	}

	if err := store.BindCredential(ctx, cred); err != nil {
		t.Fatal(err)
	}

	err := store.BindCredential(ctx, cred)
	if err != identity.ErrDuplicateCredential {
		t.Fatalf("expected ErrDuplicateCredential, got %v", err)
	}
}

func TestMockStoreBindMissingSubject(t *testing.T) {
	store := setupMockStore(t)
	ctx := context.Background()

	cred := &identity.Credential{
		SubjectID:      identity.SubjectIDFromInt64(999),
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "admin",
		CredentialData: "hash",
	}

	err := store.BindCredential(ctx, cred)
	if err != identity.ErrSubjectNotFound {
		t.Fatalf("expected ErrSubjectNotFound, got %v", err)
	}
}

func TestMockStoreFind(t *testing.T) {
	store := setupMockStore(t)
	ctx := context.Background()

	id, _ := store.CreateSubject(ctx)
	cred := &identity.Credential{
		SubjectID:      id,
		Realm:          "users",
		IdentityType:   identity.TypePassword,
		Identifier:     "admin",
		CredentialData: "hash",
	}
	store.BindCredential(ctx, cred)

	found, err := store.FindByRealmTypeIdentifier(ctx, "users", identity.TypePassword, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if found.SubjectID != id {
		t.Fatalf("expected subjectID %s, got %s", id, found.SubjectID)
	}

	_, err = store.FindByRealmTypeIdentifier(ctx, "users", identity.TypePassword, "nonexistent")
	if err != identity.ErrCredentialNotFound {
		t.Fatalf("expected ErrCredentialNotFound, got %v", err)
	}
}

func TestMockStoreList(t *testing.T) {
	store := setupMockStore(t)
	ctx := context.Background()

	id, _ := store.CreateSubject(ctx)
	store.BindCredential(ctx, &identity.Credential{
		SubjectID: id, Realm: "users", IdentityType: identity.TypePassword, Identifier: "admin", CredentialData: "hash",
	})
	store.BindCredential(ctx, &identity.Credential{
		SubjectID: id, Realm: "users", IdentityType: identity.TypeEmail, Identifier: "a@b.com", CredentialData: "",
	})

	list, err := store.ListBySubjectRealm(ctx, id, "users")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(list))
	}
}

func TestMockStoreFindReturnsIsActive(t *testing.T) {
	store := setupMockStore(t)
	ctx := context.Background()

	id, _ := store.CreateSubject(ctx)
	store.BindCredential(ctx, &identity.Credential{
		SubjectID: id, Realm: "users", IdentityType: identity.TypePassword, Identifier: "active_user", CredentialData: "hash",
	})

	cred, err := store.FindByRealmTypeIdentifier(ctx, "users", identity.TypePassword, "active_user")
	if err != nil {
		t.Fatal(err)
	}
	if !cred.IsActive {
		t.Fatal("expected IsActive=true for active subject")
	}

	store.SetInactive(id)
	cred, err = store.FindByRealmTypeIdentifier(ctx, "users", identity.TypePassword, "active_user")
	if err != nil {
		t.Fatal(err)
	}
	if cred.IsActive {
		t.Fatal("expected IsActive=false after SetInactive")
	}
}

func TestMockStoreWithTransaction(t *testing.T) {
	store := setupMockStore(t)
	ctx := context.Background()

	var capturedID identity.SubjectID
	err := store.WithTransaction(ctx, func(ctx context.Context) error {
		id, err := store.CreateSubject(ctx)
		if err != nil {
			return err
		}
		capturedID = id
		return store.BindCredential(ctx, &identity.Credential{
			SubjectID:    id,
			Realm:        "users",
			IdentityType: identity.TypePassword,
			Identifier:   "tx_user",
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if capturedID == "" {
		t.Fatal("expected non-empty subject ID from transaction")
	}

	cred, err := store.FindByRealmTypeIdentifier(ctx, "users", identity.TypePassword, "tx_user")
	if err != nil {
		t.Fatal(err)
	}
	if cred.SubjectID != capturedID {
		t.Fatalf("expected SubjectID %s, got %s", capturedID, cred.SubjectID)
	}
}
