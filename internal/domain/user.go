package domain

import (
	"errors"
	"strings"
	"time"
)

// UserID is the stable internal identifier of a Tiny Cosmos user.
//
// It is an application-generated UUID in canonical string form and never
// depends on any external identity provider.
type UserID string

// User is a Tiny Cosmos account. External logins (Telegram, Apple, Google,
// email, phone) are linked to a User through UserIdentity values; they are not
// the account itself.
type User struct {
	ID        UserID
	CreatedAt time.Time // UTC
}

var (
	ErrUserIDRequired        = errors.New("domain: user id is required")
	ErrUserCreatedAtRequired = errors.New("domain: user created_at is required")
)

// Validate reports whether the user has the fields required by the domain.
func (u User) Validate() error {
	if strings.TrimSpace(string(u.ID)) == "" {
		return ErrUserIDRequired
	}
	if u.CreatedAt.IsZero() {
		return ErrUserCreatedAtRequired
	}
	return nil
}
