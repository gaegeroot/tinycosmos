package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/gaegeroot/tinycosmos/internal/domain"
)

func telegramGroup(gid domain.GroupID, iid domain.GroupIntegrationID, chatID string) (domain.Group, domain.GroupIntegration) {
	return domain.Group{ID: gid, CreatedAt: testTime},
		domain.GroupIntegration{ID: iid, GroupID: gid, Provider: domain.IntegrationProviderTelegram, ExternalGroupID: chatID}
}

func TestCreateUserWithIdentity(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	u := domain.User{ID: "user-1", CreatedAt: testTime}
	if err := r.CreateUserWithIdentity(ctx, u, telegramIdentity("ident-1", "user-1", "123")); err != nil {
		t.Fatalf("CreateUserWithIdentity: %v", err)
	}
	if _, err := r.GetUserByID(ctx, "user-1"); err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	got, err := r.GetUserIdentityByProvider(ctx, domain.IdentityProviderTelegram, "123")
	if err != nil || got.UserID != "user-1" {
		t.Fatalf("GetUserIdentityByProvider = %+v, %v", got, err)
	}
}

func TestCreateUserWithIdentityRollsBackOnDuplicateIdentity(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	if err := r.CreateUserWithIdentity(ctx, domain.User{ID: "user-1", CreatedAt: testTime},
		telegramIdentity("ident-1", "user-1", "123")); err != nil {
		t.Fatalf("first: %v", err)
	}

	err := r.CreateUserWithIdentity(ctx, domain.User{ID: "user-2", CreatedAt: testTime},
		telegramIdentity("ident-2", "user-2", "123"))
	if !errors.Is(err, domain.ErrUserIdentityAlreadyExists) {
		t.Fatalf("duplicate = %v, want ErrUserIdentityAlreadyExists", err)
	}
	if _, err := r.GetUserByID(ctx, "user-2"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("user-2 should not exist after rollback, got %v", err)
	}
}

func TestCreateUserWithIdentityMismatchedUser(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	err := r.CreateUserWithIdentity(context.Background(), domain.User{ID: "user-1", CreatedAt: testTime},
		telegramIdentity("ident-1", "someone-else", "123"))
	if err == nil {
		t.Fatal("want error for mismatched identity user id")
	}
}

func TestCreateAndGetGroupIntegration(t *testing.T) {
	r := NewGroupRepository(newTestDB(t))
	ctx := context.Background()

	g, gi := telegramGroup("group-1", "gi-1", "-100123")
	if err := r.CreateGroupWithIntegration(ctx, g, gi); err != nil {
		t.Fatalf("CreateGroupWithIntegration: %v", err)
	}
	got, err := r.GetGroupIntegration(ctx, domain.IntegrationProviderTelegram, "-100123")
	if err != nil {
		t.Fatalf("GetGroupIntegration: %v", err)
	}
	if got != gi {
		t.Fatalf("GetGroupIntegration = %+v, want %+v", got, gi)
	}
}

func TestCreateGroupDuplicateExternalGroupRollsBack(t *testing.T) {
	db := newTestDB(t)
	r := NewGroupRepository(db)
	ctx := context.Background()

	g1, gi1 := telegramGroup("group-1", "gi-1", "-100123")
	if err := r.CreateGroupWithIntegration(ctx, g1, gi1); err != nil {
		t.Fatalf("first: %v", err)
	}
	g2, gi2 := telegramGroup("group-2", "gi-2", "-100123")
	err := r.CreateGroupWithIntegration(ctx, g2, gi2)
	if !errors.Is(err, domain.ErrGroupIntegrationAlreadyExists) {
		t.Fatalf("duplicate = %v, want ErrGroupIntegrationAlreadyExists", err)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM groups WHERE id = 'group-2'`).Scan(&n)
	if n != 0 {
		t.Fatal("group-2 was stored despite failed integration insert")
	}
}

func TestCreateGroupDuplicateGroupID(t *testing.T) {
	r := NewGroupRepository(newTestDB(t))
	ctx := context.Background()

	g, gi := telegramGroup("group-1", "gi-1", "-1")
	if err := r.CreateGroupWithIntegration(ctx, g, gi); err != nil {
		t.Fatal(err)
	}
	g2, gi2 := telegramGroup("group-1", "gi-2", "-2")
	if err := r.CreateGroupWithIntegration(ctx, g2, gi2); !errors.Is(err, domain.ErrGroupAlreadyExists) {
		t.Fatalf("got %v, want ErrGroupAlreadyExists", err)
	}
}

func TestGetGroupIntegrationNotFoundAndUnknownProvider(t *testing.T) {
	r := NewGroupRepository(newTestDB(t))
	ctx := context.Background()

	if _, err := r.GetGroupIntegration(ctx, domain.IntegrationProviderTelegram, "-1"); !errors.Is(err, domain.ErrGroupIntegrationNotFound) {
		t.Fatalf("missing = %v, want ErrGroupIntegrationNotFound", err)
	}
	if _, err := r.GetGroupIntegration(ctx, "slack", "-1"); !errors.Is(err, domain.ErrUnknownIntegrationProvider) {
		t.Fatalf("unknown provider = %v, want ErrUnknownIntegrationProvider", err)
	}
}

func TestMembershipLifecycle(t *testing.T) {
	db := newTestDB(t)
	users, groups, members := NewUserRepository(db), NewGroupRepository(db), NewMembershipRepository(db)
	ctx := context.Background()

	mustCreateUser(t, users, "user-1")
	g, gi := telegramGroup("group-1", "gi-1", "-1")
	if err := groups.CreateGroupWithIntegration(ctx, g, gi); err != nil {
		t.Fatal(err)
	}

	status := func() domain.MembershipStatus {
		t.Helper()
		s, err := members.GetMembershipStatus(ctx, "group-1", "user-1")
		if err != nil {
			t.Fatalf("GetMembershipStatus: %v", err)
		}
		return s
	}
	step := func(name string, fn func(context.Context, domain.GroupID, domain.UserID) (bool, error), wantChanged bool, wantStatus domain.MembershipStatus) {
		t.Helper()
		changed, err := fn(ctx, "group-1", "user-1")
		if err != nil || changed != wantChanged {
			t.Fatalf("%s = %v, %v; want %v, nil", name, changed, err, wantChanged)
		}
		if s := status(); s != wantStatus {
			t.Fatalf("after %s: status %q, want %q", name, s, wantStatus)
		}
	}

	if _, err := members.GetMembershipStatus(ctx, "group-1", "user-1"); !errors.Is(err, domain.ErrMembershipNotFound) {
		t.Fatalf("before joining = %v, want ErrMembershipNotFound", err)
	}
	if changed, err := members.MarkMembershipLeft(ctx, "group-1", "user-1"); err != nil || changed {
		t.Fatalf("leave before joining = %v, %v; want false, nil", changed, err)
	}
	if _, err := members.GetMembershipStatus(ctx, "group-1", "user-1"); !errors.Is(err, domain.ErrMembershipNotFound) {
		t.Fatal("leaving before joining must not create a row")
	}

	step("join", members.ActivateMembership, true, domain.MembershipStatusActive)
	step("join again", members.ActivateMembership, false, domain.MembershipStatusActive)
	step("leave", members.MarkMembershipLeft, true, domain.MembershipStatusLeft)
	step("leave again", members.MarkMembershipLeft, false, domain.MembershipStatusLeft)
	step("rejoin", members.ActivateMembership, true, domain.MembershipStatusActive)

	var rows int
	db.QueryRow(`SELECT count(*) FROM group_memberships`).Scan(&rows)
	if rows != 1 {
		t.Fatalf("membership rows = %d, want 1", rows)
	}
}

func TestActivateMembershipMissingReferences(t *testing.T) {
	db := newTestDB(t)
	users, members := NewUserRepository(db), NewMembershipRepository(db)
	mustCreateUser(t, users, "user-1")

	_, err := members.ActivateMembership(context.Background(), "no-such-group", "user-1")
	if !errors.Is(err, domain.ErrMembershipReferenceNotFound) {
		t.Fatalf("got %v, want ErrMembershipReferenceNotFound", err)
	}
}

func TestUpdateExternalGroupID(t *testing.T) {
	r := NewGroupRepository(newTestDB(t))
	ctx := context.Background()

	g, gi := telegramGroup("group-1", "gi-1", "-1")
	if err := r.CreateGroupWithIntegration(ctx, g, gi); err != nil {
		t.Fatal(err)
	}
	g2, gi2 := telegramGroup("group-2", "gi-2", "-2")
	if err := r.CreateGroupWithIntegration(ctx, g2, gi2); err != nil {
		t.Fatal(err)
	}

	moved, err := r.UpdateExternalGroupID(ctx, domain.IntegrationProviderTelegram, "-1", "-1001")
	if err != nil || !moved {
		t.Fatalf("move = %v, %v", moved, err)
	}
	got, err := r.GetGroupIntegration(ctx, domain.IntegrationProviderTelegram, "-1001")
	if err != nil || got.GroupID != "group-1" {
		t.Fatalf("after move = %+v, %v", got, err)
	}
	if _, err := r.GetGroupIntegration(ctx, domain.IntegrationProviderTelegram, "-1"); !errors.Is(err, domain.ErrGroupIntegrationNotFound) {
		t.Fatalf("old id still connected: %v", err)
	}

	// Moving again is a no-op.
	if moved, err := r.UpdateExternalGroupID(ctx, domain.IntegrationProviderTelegram, "-1", "-1001"); err != nil || moved {
		t.Fatalf("second move = %v, %v; want false, nil", moved, err)
	}
	// Moving onto an ID that's already connected fails.
	if _, err := r.UpdateExternalGroupID(ctx, domain.IntegrationProviderTelegram, "-2", "-1001"); !errors.Is(err, domain.ErrGroupIntegrationAlreadyExists) {
		t.Fatalf("collision = %v, want ErrGroupIntegrationAlreadyExists", err)
	}
}
