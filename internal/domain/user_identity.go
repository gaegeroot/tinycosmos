package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// UserIdentityID is the internal identifier of a linked external identity.
// It is an application-generated UUID in canonical string form.
type UserIdentityID string

// IdentityProvider names an external system through which a user can
// authenticate. Its string value is the canonical, persisted representation.
type IdentityProvider string

// Supported identity providers.
const (
	IdentityProviderTelegram IdentityProvider = "telegram"
	IdentityProviderEmail    IdentityProvider = "email"
	IdentityProviderApple    IdentityProvider = "apple"
	IdentityProviderGoogle   IdentityProvider = "google"
	IdentityProviderPhone    IdentityProvider = "phone"
)

// IdentityProviders returns all supported identity providers.
func IdentityProviders() []IdentityProvider {
	return []IdentityProvider{
		IdentityProviderTelegram,
		IdentityProviderEmail,
		IdentityProviderApple,
		IdentityProviderGoogle,
		IdentityProviderPhone,
	}
}

// ErrUnknownIdentityProvider is returned for provider values that are not
// supported.
var ErrUnknownIdentityProvider = errors.New("domain: unknown identity provider")

// Valid reports whether p is a supported identity provider.
func (p IdentityProvider) Valid() bool {
	switch p {
	case IdentityProviderTelegram,
		IdentityProviderEmail,
		IdentityProviderApple,
		IdentityProviderGoogle,
		IdentityProviderPhone:
		return true
	}
	return false
}

// String returns the canonical string value of p.
func (p IdentityProvider) String() string { return string(p) }

// ParseIdentityProvider converts a canonical provider value into an
// IdentityProvider. Matching is exact; values are not case-folded or trimmed.
func ParseIdentityProvider(s string) (IdentityProvider, error) {
	p := IdentityProvider(s)
	if !p.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnknownIdentityProvider, s)
	}
	return p, nil
}

// UserIdentity links an external provider identity to a Tiny Cosmos user.
//
// The pair (Provider, ProviderUserID) identifies exactly one external identity
// and may belong to only one user. ProviderUserID must be the provider's stable
// identifier (e.g. Telegram user ID, Apple/Google subject, normalized email),
// never a display name.
type UserIdentity struct {
	ID             UserIdentityID
	UserID         UserID
	Provider       IdentityProvider
	ProviderUserID string
	CreatedAt      time.Time // UTC; when the identity was linked
}

var (
	ErrUserIdentityIDRequired        = errors.New("domain: user identity id is required")
	ErrUserIdentityUserIDRequired    = errors.New("domain: user identity user id is required")
	ErrProviderUserIDRequired        = errors.New("domain: provider user id is required")
	ErrUserIdentityCreatedAtRequired = errors.New("domain: user identity created_at is required")
)

// Validate reports whether the identity has the fields required by the domain.
func (i UserIdentity) Validate() error {
	if strings.TrimSpace(string(i.ID)) == "" {
		return ErrUserIdentityIDRequired
	}
	if strings.TrimSpace(string(i.UserID)) == "" {
		return ErrUserIdentityUserIDRequired
	}
	if !i.Provider.Valid() {
		return fmt.Errorf("%w: %q", ErrUnknownIdentityProvider, string(i.Provider))
	}
	if strings.TrimSpace(i.ProviderUserID) == "" {
		return ErrProviderUserIDRequired
	}
	if i.CreatedAt.IsZero() {
		return ErrUserIdentityCreatedAtRequired
	}
	return nil
}
