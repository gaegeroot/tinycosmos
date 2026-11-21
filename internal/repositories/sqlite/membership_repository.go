package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gaegeroot/tinycosmos/internal/domain"
	sqlite3 "modernc.org/sqlite/lib"
)

// MembershipRepository persists group memberships. Rows are never deleted:
// leaving sets the status to left and rejoining sets it back to active.
type MembershipRepository struct {
	db *sql.DB
}

// NewMembershipRepository returns a MembershipRepository backed by db.
func NewMembershipRepository(db *sql.DB) *MembershipRepository {
	return &MembershipRepository{db: db}
}

// ActivateMembership makes the user an active member of the group, creating
// the membership row if needed or reactivating a left one.
//
// It reports whether anything changed: false means the user was already an
// active member. It returns domain.ErrMembershipReferenceNotFound if the
// group or user does not exist.
func (r *MembershipRepository) ActivateMembership(ctx context.Context, groupID domain.GroupID, userID domain.UserID) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO group_memberships (group_id, user_id, status)
		 VALUES (?, ?, ?)
		 ON CONFLICT (group_id, user_id) DO UPDATE
		    SET status = excluded.status
		  WHERE group_memberships.status <> excluded.status`,
		string(groupID), string(userID), string(domain.MembershipStatusActive),
	)
	if err != nil {
		if constraintCode(err) == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY {
			return false, fmt.Errorf("activate membership of user %q in group %q: %w",
				userID, groupID, domain.ErrMembershipReferenceNotFound)
		}
		return false, fmt.Errorf("activate membership of user %q in group %q: %w", userID, groupID, err)
	}
	return rowsChanged(res)
}

// MarkMembershipLeft sets an active membership to left. It reports whether
// anything changed: false means there was no active membership (the user
// never joined, or already left), and no row is created in that case.
func (r *MembershipRepository) MarkMembershipLeft(ctx context.Context, groupID domain.GroupID, userID domain.UserID) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE group_memberships SET status = ?
		  WHERE group_id = ? AND user_id = ? AND status = ?`,
		string(domain.MembershipStatusLeft), string(groupID), string(userID), string(domain.MembershipStatusActive),
	)
	if err != nil {
		return false, fmt.Errorf("mark user %q left group %q: %w", userID, groupID, err)
	}
	return rowsChanged(res)
}

// GetMembershipStatus returns the user's membership status in the group, or
// domain.ErrMembershipNotFound if the user has never been a member.
func (r *MembershipRepository) GetMembershipStatus(ctx context.Context, groupID domain.GroupID, userID domain.UserID) (domain.MembershipStatus, error) {
	var raw string
	err := r.db.QueryRowContext(ctx,
		`SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?`,
		string(groupID), string(userID),
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("get membership of user %q in group %q: %w", userID, groupID, domain.ErrMembershipNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("get membership of user %q in group %q: %w", userID, groupID, err)
	}
	return domain.ParseMembershipStatus(raw)
}

func rowsChanged(res sql.Result) (bool, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("sqlite: rows affected: %w", err)
	}
	return n > 0, nil
}
