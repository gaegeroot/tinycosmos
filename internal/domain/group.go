package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// GroupID is the stable internal identifier of a Tiny Cosmos group.
// It is an application-generated UUID in canonical string form.
type GroupID string

// Group is a native Tiny Cosmos group. External platforms such as Telegram
// are connected to it through GroupIntegration values; a group never depends
// on any single platform.
type Group struct {
	ID        GroupID
	CreatedAt time.Time // UTC
}

var (
	ErrGroupIDRequired        = errors.New("domain: group id is required")
	ErrGroupCreatedAtRequired = errors.New("domain: group created_at is required")
)

// Validate reports whether the group has the fields required by the domain.
func (g Group) Validate() error {
	if strings.TrimSpace(string(g.ID)) == "" {
		return ErrGroupIDRequired
	}
	if g.CreatedAt.IsZero() {
		return ErrGroupCreatedAtRequired
	}
	return nil
}

// GroupIntegrationID is the internal identifier of a group integration.
type GroupIntegrationID string

// IntegrationProvider names an external platform a group can be connected
// to. Its string value is the canonical, persisted representation.
type IntegrationProvider string

// Supported integration providers.
const (
	IntegrationProviderTelegram IntegrationProvider = "telegram"
)

// ErrUnknownIntegrationProvider is returned for unsupported provider values.
var ErrUnknownIntegrationProvider = errors.New("domain: unknown integration provider")

// Valid reports whether p is a supported integration provider.
func (p IntegrationProvider) Valid() bool {
	return p == IntegrationProviderTelegram
}

// String returns the canonical string value of p.
func (p IntegrationProvider) String() string { return string(p) }

// ParseIntegrationProvider converts a canonical value into an
// IntegrationProvider. Matching is exact.
func ParseIntegrationProvider(s string) (IntegrationProvider, error) {
	p := IntegrationProvider(s)
	if !p.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnknownIntegrationProvider, s)
	}
	return p, nil
}

// GroupIntegration connects a Tiny Cosmos group to a group on an external
// platform. The pair (Provider, ExternalGroupID) is connected to at most one
// Tiny Cosmos group.
type GroupIntegration struct {
	ID              GroupIntegrationID
	GroupID         GroupID
	Provider        IntegrationProvider
	ExternalGroupID string // e.g. the Telegram chat ID in decimal
}

var (
	ErrGroupIntegrationIDRequired      = errors.New("domain: group integration id is required")
	ErrGroupIntegrationGroupIDRequired = errors.New("domain: group integration group id is required")
	ErrExternalGroupIDRequired         = errors.New("domain: external group id is required")
)

// Validate reports whether the integration has the fields required by the
// domain.
func (i GroupIntegration) Validate() error {
	if strings.TrimSpace(string(i.ID)) == "" {
		return ErrGroupIntegrationIDRequired
	}
	if strings.TrimSpace(string(i.GroupID)) == "" {
		return ErrGroupIntegrationGroupIDRequired
	}
	if !i.Provider.Valid() {
		return fmt.Errorf("%w: %q", ErrUnknownIntegrationProvider, string(i.Provider))
	}
	if strings.TrimSpace(i.ExternalGroupID) == "" {
		return ErrExternalGroupIDRequired
	}
	return nil
}
