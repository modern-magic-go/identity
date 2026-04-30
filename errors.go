package identity

import "errors"

var (
	ErrInvalidCredential   = errors.New("identity: invalid credential")
	ErrAccountLocked       = errors.New("identity: account locked")
	ErrDuplicateCredential = errors.New("identity: duplicate credential already exists in realm")
	ErrCredentialNotFound  = errors.New("identity: credential not found")
	ErrSubjectNotFound     = errors.New("identity: subject not found")
)
