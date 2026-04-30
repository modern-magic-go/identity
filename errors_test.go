package identity

import (
	"errors"
	"testing"
)

func TestErrorSentinelsAreDistinct(t *testing.T) {
	sentinels := []error{
		ErrInvalidCredential,
		ErrAccountLocked,
		ErrDuplicateCredential,
		ErrCredentialNotFound,
		ErrSubjectNotFound,
	}

	for i, a := range sentinels {
		if !errors.Is(a, a) {
			t.Fatalf("expected errors.Is(%v, %v) to be true", a, a)
		}
		for j, b := range sentinels {
			if i != j && errors.Is(a, b) {
				t.Fatalf("expected errors.Is(%v, %v) to be false", a, b)
			}
		}
	}
}
