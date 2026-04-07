package identity

import "errors"

var (
	ErrNilSubject            = errors.New("subject is nil")
	ErrSubjectNotLoginable   = errors.New("subject is not loginable")
	ErrIdentityUnavailable   = errors.New("identity is unavailable")
	ErrCredentialUnavailable = errors.New("credential is unavailable")
	ErrUnsupportedRealm      = errors.New("unsupported realm")
	ErrUnsupportedProvider   = errors.New("unsupported provider")
	ErrUnsupportedIdentity   = errors.New("unsupported identity type")
	ErrInvalidIdentifier     = errors.New("invalid identifier")
	ErrInvalidRepository     = errors.New("repository is nil")
)
