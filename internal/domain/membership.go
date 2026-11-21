package domain

import (
	"errors"
	"fmt"
)

// MembershipStatus is the current state of a user's membership in a group.
// Membership rows are retained when people leave; only the status changes.
type MembershipStatus string

// Supported membership statuses.
const (
	MembershipStatusActive MembershipStatus = "active"
	MembershipStatusLeft   MembershipStatus = "left"
)

// ErrUnknownMembershipStatus is returned for unsupported status values.
var ErrUnknownMembershipStatus = errors.New("domain: unknown membership status")

// Valid reports whether s is a supported membership status.
func (s MembershipStatus) Valid() bool {
	return s == MembershipStatusActive || s == MembershipStatusLeft
}

// String returns the canonical string value of s.
func (s MembershipStatus) String() string { return string(s) }

// ParseMembershipStatus converts a canonical value into a MembershipStatus.
func ParseMembershipStatus(s string) (MembershipStatus, error) {
	st := MembershipStatus(s)
	if !st.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnknownMembershipStatus, s)
	}
	return st, nil
}
