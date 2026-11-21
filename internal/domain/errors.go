package domain

import "errors"

// Errors reported by persistence implementations. They live in the domain so
// callers can test for them with errors.Is without importing any storage
// package.
var (
	// ErrUserNotFound is returned when a referenced user does not exist.
	ErrUserNotFound = errors.New("domain: user not found")

	// ErrUserAlreadyExists is returned when a user with the same ID exists.
	ErrUserAlreadyExists = errors.New("domain: user already exists")

	// ErrUserIdentityNotFound is returned when no identity matches a lookup.
	ErrUserIdentityNotFound = errors.New("domain: user identity not found")

	// ErrUserIdentityAlreadyExists is returned when an identity cannot be
	// stored because its (Provider, ProviderUserID) pair is already linked to
	// a user, or because its ID is already in use.
	ErrUserIdentityAlreadyExists = errors.New("domain: user identity already exists")
)
