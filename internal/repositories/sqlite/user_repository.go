package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gaegeroot/tinycosmos/internal/domain"
	sqlite3 "modernc.org/sqlite/lib"
)

// UserRepository persists Tiny Cosmos users and their external identities.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository returns a UserRepository backed by db. The caller owns db
// and should open it with Open so that foreign keys are enforced.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser stores a new user.
//
// It returns the domain validation error if u is invalid, and
// domain.ErrUserAlreadyExists if a user with the same ID exists.
func (r *UserRepository) CreateUser(ctx context.Context, u domain.User) error {
	if err := u.Validate(); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, created_at) VALUES (?, ?)`,
		string(u.ID), formatTimestamp(u.CreatedAt),
	)
	if err != nil {
		switch constraintCode(err) {
		case sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3.SQLITE_CONSTRAINT_UNIQUE:
			return fmt.Errorf("create user %q: %w", u.ID, domain.ErrUserAlreadyExists)
		}
		return fmt.Errorf("create user %q: %w", u.ID, err)
	}
	return nil
}

// GetUserByID returns the user with the given ID, or domain.ErrUserNotFound.
func (r *UserRepository) GetUserByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	var (
		rawID, rawCreated string
	)
	err := r.db.QueryRowContext(ctx,
		`SELECT id, created_at FROM users WHERE id = ?`,
		string(id),
	).Scan(&rawID, &rawCreated)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, fmt.Errorf("get user %q: %w", id, domain.ErrUserNotFound)
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user %q: %w", id, err)
	}

	createdAt, err := parseTimestamp(rawCreated)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user %q: %w", id, err)
	}
	return domain.User{ID: domain.UserID(rawID), CreatedAt: createdAt}, nil
}

// CreateUserIdentity links an external identity to an existing user.
//
// It returns the domain validation error if identity is invalid,
// domain.ErrUserNotFound if identity.UserID does not reference an existing
// user, and domain.ErrUserIdentityAlreadyExists if the (provider,
// provider user ID) pair is already linked or the identity ID is taken.
func (r *UserRepository) CreateUserIdentity(ctx context.Context, identity domain.UserIdentity) error {
	if err := identity.Validate(); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_identities (id, user_id, provider, provider_user_id, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		string(identity.ID),
		string(identity.UserID),
		string(identity.Provider),
		identity.ProviderUserID,
		formatTimestamp(identity.CreatedAt),
	)
	if err != nil {
		switch constraintCode(err) {
		case sqlite3.SQLITE_CONSTRAINT_UNIQUE:
			return fmt.Errorf("create identity %s/%q: %w",
				identity.Provider, identity.ProviderUserID, domain.ErrUserIdentityAlreadyExists)
		case sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
			return fmt.Errorf("create identity: id %q in use: %w",
				identity.ID, domain.ErrUserIdentityAlreadyExists)
		case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
			return fmt.Errorf("create identity %s/%q for user %q: %w",
				identity.Provider, identity.ProviderUserID, identity.UserID, domain.ErrUserNotFound)
		}
		return fmt.Errorf("create identity %s/%q: %w", identity.Provider, identity.ProviderUserID, err)
	}
	return nil
}

// GetUserIdentityByProvider returns the identity for the given provider and
// provider-assigned user ID.
//
// It returns domain.ErrUnknownIdentityProvider for an unsupported provider and
// domain.ErrUserIdentityNotFound if no identity matches. providerUserID is
// matched exactly; callers are responsible for any normalization.
func (r *UserRepository) GetUserIdentityByProvider(
	ctx context.Context,
	provider domain.IdentityProvider,
	providerUserID string,
) (domain.UserIdentity, error) {
	if !provider.Valid() {
		return domain.UserIdentity{}, fmt.Errorf("%w: %q", domain.ErrUnknownIdentityProvider, string(provider))
	}

	var rawID, rawUserID, rawProvider, rawProviderUserID, rawCreated string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, provider, provider_user_id, created_at
		   FROM user_identities
		  WHERE provider = ? AND provider_user_id = ?`,
		string(provider), providerUserID,
	).Scan(&rawID, &rawUserID, &rawProvider, &rawProviderUserID, &rawCreated)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UserIdentity{}, fmt.Errorf("get identity %s/%q: %w",
			provider, providerUserID, domain.ErrUserIdentityNotFound)
	}
	if err != nil {
		return domain.UserIdentity{}, fmt.Errorf("get identity %s/%q: %w", provider, providerUserID, err)
	}

	storedProvider, err := domain.ParseIdentityProvider(rawProvider)
	if err != nil {
		return domain.UserIdentity{}, fmt.Errorf("get identity %s/%q: %w", provider, providerUserID, err)
	}
	createdAt, err := parseTimestamp(rawCreated)
	if err != nil {
		return domain.UserIdentity{}, fmt.Errorf("get identity %s/%q: %w", provider, providerUserID, err)
	}

	return domain.UserIdentity{
		ID:             domain.UserIdentityID(rawID),
		UserID:         domain.UserID(rawUserID),
		Provider:       storedProvider,
		ProviderUserID: rawProviderUserID,
		CreatedAt:      createdAt,
	}, nil
}
