// Package application holds Tiny Cosmos business rules. It coordinates
// domain models and persistence through small interfaces and never imports
// storage drivers or platform SDKs.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/gaegeroot/tinycosmos/internal/domain"
	"github.com/google/uuid"
)

// UserStore persists users and their external identities.
type UserStore interface {
	GetUserIdentityByProvider(ctx context.Context, provider domain.IdentityProvider, providerUserID string) (domain.UserIdentity, error)
	CreateUserWithIdentity(ctx context.Context, u domain.User, identity domain.UserIdentity) error
}

// GroupStore persists groups and their external platform integrations.
type GroupStore interface {
	CreateGroupWithIntegration(ctx context.Context, g domain.Group, gi domain.GroupIntegration) error
	GetGroupIntegration(ctx context.Context, provider domain.IntegrationProvider, externalGroupID string) (domain.GroupIntegration, error)
	UpdateExternalGroupID(ctx context.Context, provider domain.IntegrationProvider, oldExternalGroupID, newExternalGroupID string) (bool, error)
}

// MembershipStore persists group memberships.
type MembershipStore interface {
	ActivateMembership(ctx context.Context, groupID domain.GroupID, userID domain.UserID) (changed bool, err error)
	MarkMembershipLeft(ctx context.Context, groupID domain.GroupID, userID domain.UserID) (changed bool, err error)
}

// Clock returns the current time. Services convert it to UTC.
type Clock func() time.Time

// IDGenerator returns a new unique identifier.
type IDGenerator func() string

// Application errors. Adapters translate these into user-facing messages.
var (
	// ErrNotGroupAdmin is returned when someone who is not an admin of the
	// external group tries an admin-only action.
	ErrNotGroupAdmin = errors.New("application: only a group admin can do that")

	// ErrGroupNotActivated is returned when an action needs a Tiny Cosmos
	// group but the external group has not been activated.
	ErrGroupNotActivated = errors.New("application: group has not activated Tiny Cosmos")
)

func defaultClock() time.Time { return time.Now() }

func defaultIDGenerator() string { return uuid.NewString() }
