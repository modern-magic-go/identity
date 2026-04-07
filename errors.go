package identity

import "errors"

var (
	ErrNilSubject            = errors.New("subject is nil")
	ErrSubjectNotLoginable   = errors.New("subject is not loginable")
	ErrIdentityUnavailable   = errors.New("identity is unavailable")
	ErrCredentialUnavailable = errors.New("credential is unavailable")
	ErrChallengeUnavailable  = errors.New("verification challenge is unavailable")
	ErrUnsupportedRealm      = errors.New("unsupported realm")
	ErrUnsupportedProvider   = errors.New("unsupported provider")
	ErrUnsupportedIdentity   = errors.New("unsupported identity type")
	ErrInvalidIdentifier     = errors.New("invalid identifier")
	ErrEmptyChallengeCode    = errors.New("verification code is empty")
	ErrInvalidSessionManager = errors.New("session manager is nil")
	ErrInvalidRepository     = errors.New("repository is nil")
)
