package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gaegeroot/tinycosmos/internal/domain"
	sqlite3 "modernc.org/sqlite/lib"
)

// GroupRepository persists Tiny Cosmos groups and their external platform
// integrations.
type GroupRepository struct {
	db *sql.DB
}

// NewGroupRepository returns a GroupRepository backed by db.
func NewGroupRepository(db *sql.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// CreateGroupWithIntegration stores a new group and its first integration
// atomically: either both are stored or neither is.
//
// It returns domain.ErrGroupIntegrationAlreadyExists if the external group is
// already connected to a Tiny Cosmos group (for example, a concurrent
// activation won the race), and domain.ErrGroupAlreadyExists if the group ID
// is taken. In both cases nothing is stored.
func (r *GroupRepository) CreateGroupWithIntegration(ctx context.Context, g domain.Group, gi domain.GroupIntegration) error {
	if err := g.Validate(); err != nil {
		return err
	}
	if err := gi.Validate(); err != nil {
		return err
	}
	if gi.GroupID != g.ID {
		return fmt.Errorf("create group with integration: integration group id %q does not match group id %q",
			gi.GroupID, g.ID)
	}

	return inTx(ctx, r.db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO groups (id, created_at) VALUES (?, ?)`,
			string(g.ID), formatTimestamp(g.CreatedAt),
		)
		if err != nil {
			switch constraintCode(err) {
			case sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3.SQLITE_CONSTRAINT_UNIQUE:
				return fmt.Errorf("create group %q: %w", g.ID, domain.ErrGroupAlreadyExists)
			}
			return fmt.Errorf("create group %q: %w", g.ID, err)
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO group_integrations (id, group_id, provider, external_group_id)
			 VALUES (?, ?, ?, ?)`,
			string(gi.ID), string(gi.GroupID), string(gi.Provider), gi.ExternalGroupID,
		)
		if err != nil {
			switch constraintCode(err) {
			case sqlite3.SQLITE_CONSTRAINT_UNIQUE:
				return fmt.Errorf("connect %s group %q: %w",
					gi.Provider, gi.ExternalGroupID, domain.ErrGroupIntegrationAlreadyExists)
			case sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
				return fmt.Errorf("connect group: integration id %q in use: %w",
					gi.ID, domain.ErrGroupIntegrationAlreadyExists)
			}
			return fmt.Errorf("connect %s group %q: %w", gi.Provider, gi.ExternalGroupID, err)
		}
		return nil
	})
}

// GetGroupIntegration returns the integration for an external group, or
// domain.ErrGroupIntegrationNotFound if that group is not connected.
func (r *GroupRepository) GetGroupIntegration(
	ctx context.Context,
	provider domain.IntegrationProvider,
	externalGroupID string,
) (domain.GroupIntegration, error) {
	if !provider.Valid() {
		return domain.GroupIntegration{}, fmt.Errorf("%w: %q", domain.ErrUnknownIntegrationProvider, string(provider))
	}

	var id, groupID, rawProvider, extID string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, group_id, provider, external_group_id
		   FROM group_integrations
		  WHERE provider = ? AND external_group_id = ?`,
		string(provider), externalGroupID,
	).Scan(&id, &groupID, &rawProvider, &extID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.GroupIntegration{}, fmt.Errorf("get %s group %q: %w",
			provider, externalGroupID, domain.ErrGroupIntegrationNotFound)
	}
	if err != nil {
		return domain.GroupIntegration{}, fmt.Errorf("get %s group %q: %w", provider, externalGroupID, err)
	}

	p, err := domain.ParseIntegrationProvider(rawProvider)
	if err != nil {
		return domain.GroupIntegration{}, fmt.Errorf("get %s group %q: %w", provider, externalGroupID, err)
	}
	return domain.GroupIntegration{
		ID:              domain.GroupIntegrationID(id),
		GroupID:         domain.GroupID(groupID),
		Provider:        p,
		ExternalGroupID: extID,
	}, nil
}

// UpdateExternalGroupID points an integration at a new external group ID,
// for platforms that change a group's ID (Telegram does when a basic group
// is upgraded to a supergroup). It reports whether an integration was
// updated: false means no integration had the old ID.
//
// It returns domain.ErrGroupIntegrationAlreadyExists if the new ID is already
// connected to a Tiny Cosmos group.
func (r *GroupRepository) UpdateExternalGroupID(
	ctx context.Context,
	provider domain.IntegrationProvider,
	oldExternalGroupID, newExternalGroupID string,
) (bool, error) {
	if !provider.Valid() {
		return false, fmt.Errorf("%w: %q", domain.ErrUnknownIntegrationProvider, string(provider))
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE group_integrations SET external_group_id = ?
		  WHERE provider = ? AND external_group_id = ?`,
		newExternalGroupID, string(provider), oldExternalGroupID,
	)
	if err != nil {
		if constraintCode(err) == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return false, fmt.Errorf("move %s group %q to %q: %w",
				provider, oldExternalGroupID, newExternalGroupID, domain.ErrGroupIntegrationAlreadyExists)
		}
		return false, fmt.Errorf("move %s group %q to %q: %w", provider, oldExternalGroupID, newExternalGroupID, err)
	}
	return rowsChanged(res)
}
