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

	// ErrGroupAlreadyExists is returned when a group with the same ID exists.
	ErrGroupAlreadyExists = errors.New("domain: group already exists")

	// ErrGroupIntegrationNotFound is returned when no integration matches a
	// lookup.
	ErrGroupIntegrationNotFound = errors.New("domain: group integration not found")

	// ErrGroupIntegrationAlreadyExists is returned when an external group is
	// already connected to a Tiny Cosmos group, or the integration ID is taken.
	ErrGroupIntegrationAlreadyExists = errors.New("domain: group integration already exists")

	// ErrMembershipNotFound is returned when a user has never been a member
	// of a group.
	ErrMembershipNotFound = errors.New("domain: membership not found")

	// ErrMembershipReferenceNotFound is returned when a membership refers to a
	// group or user that does not exist.
	ErrMembershipReferenceNotFound = errors.New("domain: membership group or user not found")
)
