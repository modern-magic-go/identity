package crypto

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("hello123", DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(hash, "$2a$10$") {
		t.Fatalf("expected bcrypt hash prefix $2a$10$, got %s", hash[:20])
	}

	err = Verify(hash, "hello123")
	if err != nil {
		t.Fatalf("expected nil for correct password, got %v", err)
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	hash, err := Hash("correct", DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	err = Verify(hash, "wrong")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestVerifyBogusHash(t *testing.T) {
	err := Verify("not-a-valid-hash", "anything")
	if err == nil {
		t.Fatal("expected error for invalid hash format")
	}
}

func TestHashLowCost(t *testing.T) {
	hash, err := Hash("password", 3)
	if err != nil {
		t.Fatal(err)
	}

	err = Verify(hash, "password")
	if err != nil {
		t.Fatalf("expected valid hash for cost=3 (library silently uses MinCost), got %v", err)
	}
}

func TestHashEmptyPassword(t *testing.T) {
	hash, err := Hash("", DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	err = Verify(hash, "")
	if err != nil {
		t.Fatalf("expected nil for empty password match, got %v", err)
	}
}

func TestBcryptVerifierType(t *testing.T) {
	b := &Bcrypt{}
	if b.Type() != "PASSWORD" {
		t.Fatalf("expected PASSWORD type, got %s", b.Type())
	}
}

func TestBcryptVerifierVerify(t *testing.T) {
	hash, err := Hash("mypassword", DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	b := &Bcrypt{}
	ok, err := b.Verify(hash, "mypassword")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected true for correct password")
	}

	ok, err = b.Verify(hash, "wrongpassword")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false for wrong password")
	}
}
