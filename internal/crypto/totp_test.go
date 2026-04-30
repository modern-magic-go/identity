package crypto

import (
	"strings"
	"testing"
	"time"

	"github.com/modern-magic-go/identity"
	"github.com/pquerna/otp/totp"
)

func TestTOTPImplementsVerifier(t *testing.T) {
	var v CredentialVerifier = &TOTP{}
	if v.Type() != identity.TypeTOTP {
		t.Fatalf("expected Type=%s, got %s", identity.TypeTOTP, v.Type())
	}
}

func TestGenerateTOTPKey(t *testing.T) {
	secret, url, err := GenerateTOTPKey("MyApp", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}
	if len(secret) < 16 {
		t.Fatalf("expected secret length >= 16, got %d", len(secret))
	}
	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Fatalf("expected URL to start with otpauth://totp/, got %s", url)
	}
	if !strings.Contains(url, secret) {
		t.Fatal("expected URL to contain the secret")
	}
}

func TestTOTPVerifyCorrect(t *testing.T) {
	secret, _, err := GenerateTOTPKey("MyApp", "alice")
	if err != nil {
		t.Fatal(err)
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	v := &TOTP{}
	match, err := v.Verify(secret, code)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Fatal("expected Verify to return true for correct code")
	}
}

func TestTOTPVerifyWrong(t *testing.T) {
	secret, _, err := GenerateTOTPKey("MyApp", "alice")
	if err != nil {
		t.Fatal(err)
	}

	v := &TOTP{}
	match, err := v.Verify(secret, "000000")
	if err != nil {
		t.Fatal(err)
	}
	if match {
		t.Fatal("expected Verify to return false for wrong code")
	}
}
