package application_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaegeroot/tinycosmos/database"
	"github.com/gaegeroot/tinycosmos/internal/application"
	"github.com/gaegeroot/tinycosmos/internal/domain"
	"github.com/gaegeroot/tinycosmos/internal/repositories/sqlite"
)

const (
	chatID  int64 = -100123
	adminID int64 = 111
	aliceID int64 = 222
)

// fakeAdmins answers admin checks from a fixed set and counts calls.
type fakeAdmins struct {
	admins map[int64]bool
	err    error
	calls  atomic.Int32
}

func (f *fakeAdmins) IsTelegramGroupAdmin(_ context.Context, _ int64, userID int64) (bool, error) {
	f.calls.Add(1)
	return f.admins[userID], f.err
}

type env struct {
	users   *application.UserService
	groups  *application.GroupService
	admins  *fakeAdmins
	members *sqlite.MembershipRepository
}

func newEnv(t *testing.T) env {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := sqlite.Migrate(ctx, db, database.Migrations()); err != nil {
		t.Fatal(err)
	}

	var n atomic.Int64
	newID := func() string { return fmt.Sprintf("id-%d", n.Add(1)) }
	now := func() time.Time { return time.Date(2026, 11, 21, 20, 0, 0, 0, time.UTC) }

	admins := &fakeAdmins{admins: map[int64]bool{adminID: true}}
	members := sqlite.NewMembershipRepository(db)
	users := application.NewUserService(sqlite.NewUserRepository(db), now, newID)
	groups := application.NewGroupService(sqlite.NewGroupRepository(db), members, users, admins, now, newID)
	return env{users: users, groups: groups, admins: admins, members: members}
}

func (e env) status(t *testing.T, g domain.GroupID, telegramUserID int64) domain.MembershipStatus {
	t.Helper()
	uid, err := e.users.FindTelegramUser(context.Background(), telegramUserID)
	if err != nil {
		t.Fatalf("FindTelegramUser(%d): %v", telegramUserID, err)
	}
	s, err := e.members.GetMembershipStatus(context.Background(), g, uid)
	if err != nil {
		t.Fatalf("GetMembershipStatus: %v", err)
	}
	return s
}

func TestProvisionTelegramUserIsIdempotent(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	id1, created, err := e.users.ProvisionTelegramUser(ctx, aliceID)
	if err != nil || !created {
		t.Fatalf("first = %q, %v, %v; want created", id1, created, err)
	}
	id2, created, err := e.users.ProvisionTelegramUser(ctx, aliceID)
	if err != nil || created || id2 != id1 {
		t.Fatalf("second = %q, %v, %v; want %q, not created", id2, created, err, id1)
	}
}

func TestProvisionTelegramUserConcurrent(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	const n = 8
	ids := make([]domain.UserID, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids[i], _, errs[i] = e.users.ProvisionTelegramUser(ctx, aliceID)
		}()
	}
	wg.Wait()
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("call %d: %v", i, errs[i])
		}
		if ids[i] != ids[0] {
			t.Fatalf("call %d returned %q, call 0 returned %q", i, ids[i], ids[0])
		}
	}
}

func TestProvisionTelegramUserRejectsInvalidID(t *testing.T) {
	e := newEnv(t)
	if _, _, err := e.users.ProvisionTelegramUser(context.Background(), 0); err == nil {
		t.Fatal("want error for user id 0")
	}
}

func TestActivateTelegramGroupByAdmin(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	res, err := e.groups.ActivateTelegramGroup(ctx, chatID, adminID)
	if err != nil {
		t.Fatalf("ActivateTelegramGroup: %v", err)
	}
	if !res.NewlyActivated || res.GroupID == "" || res.ActivatorUserID == "" {
		t.Fatalf("result = %+v", res)
	}
	if s := e.status(t, res.GroupID, adminID); s != domain.MembershipStatusActive {
		t.Fatalf("activator status = %q, want active", s)
	}
	if active, err := e.groups.IsTelegramGroupActive(ctx, chatID); err != nil || !active {
		t.Fatalf("IsTelegramGroupActive = %v, %v", active, err)
	}
}

func TestActivateTelegramGroupByNonAdmin(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	_, err := e.groups.ActivateTelegramGroup(ctx, chatID, aliceID)
	if !errors.Is(err, application.ErrNotGroupAdmin) {
		t.Fatalf("got %v, want ErrNotGroupAdmin", err)
	}
	if active, _ := e.groups.IsTelegramGroupActive(ctx, chatID); active {
		t.Fatal("group was activated by a non-admin")
	}
	if _, err := e.users.FindTelegramUser(ctx, aliceID); !errors.Is(err, domain.ErrUserIdentityNotFound) {
		t.Fatal("a user was created for a rejected activation")
	}
}

func TestActivateTelegramGroupAdminCheckFails(t *testing.T) {
	e := newEnv(t)
	e.admins.err = errors.New("telegram unavailable")

	_, err := e.groups.ActivateTelegramGroup(context.Background(), chatID, adminID)
	if err == nil || errors.Is(err, application.ErrNotGroupAdmin) {
		t.Fatalf("got %v, want a wrapped verifier error", err)
	}
}

func TestActivateTelegramGroupTwice(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	first, err := e.groups.ActivateTelegramGroup(ctx, chatID, adminID)
	if err != nil {
		t.Fatal(err)
	}
	callsAfterFirst := e.admins.calls.Load()

	// Anyone re-sending /start gets the existing group, with no admin check.
	again, err := e.groups.ActivateTelegramGroup(ctx, chatID, aliceID)
	if err != nil {
		t.Fatalf("second activation: %v", err)
	}
	if again.NewlyActivated || again.GroupID != first.GroupID || again.ActivatorUserID != "" {
		t.Fatalf("second result = %+v, want existing group %q, not newly activated", again, first.GroupID)
	}
	if e.admins.calls.Load() != callsAfterFirst {
		t.Fatal("admin check ran for an already-active group")
	}
	if _, err := e.users.FindTelegramUser(ctx, aliceID); !errors.Is(err, domain.ErrUserIdentityNotFound) {
		t.Fatal("re-sending /start must not enroll the sender")
	}
}

func TestActivateTelegramGroupConcurrent(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	const n = 6
	results := make([]application.ActivationResult, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = e.groups.ActivateTelegramGroup(ctx, chatID, adminID)
		}()
	}
	wg.Wait()

	newly := 0
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("call %d: %v", i, errs[i])
		}
		if results[i].GroupID != results[0].GroupID {
			t.Fatalf("calls disagree on group: %q vs %q", results[i].GroupID, results[0].GroupID)
		}
		if results[i].NewlyActivated {
			newly++
		}
	}
	if newly != 1 {
		t.Fatalf("%d calls reported NewlyActivated, want exactly 1", newly)
	}
}

func TestEnrollTelegramMember(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	if _, err := e.groups.EnrollTelegramMember(ctx, chatID, aliceID); !errors.Is(err, application.ErrGroupNotActivated) {
		t.Fatalf("before activation = %v, want ErrGroupNotActivated", err)
	}

	act, err := e.groups.ActivateTelegramGroup(ctx, chatID, adminID)
	if err != nil {
		t.Fatal(err)
	}

	res, err := e.groups.EnrollTelegramMember(ctx, chatID, aliceID)
	if err != nil || !res.NewlyEnrolled || res.GroupID != act.GroupID {
		t.Fatalf("enroll = %+v, %v", res, err)
	}
	if s := e.status(t, act.GroupID, aliceID); s != domain.MembershipStatusActive {
		t.Fatalf("status = %q, want active", s)
	}

	again, err := e.groups.EnrollTelegramMember(ctx, chatID, aliceID)
	if err != nil || again.NewlyEnrolled || again.UserID != res.UserID {
		t.Fatalf("enroll again = %+v, %v; want same user, not newly enrolled", again, err)
	}

	// The activator is already enrolled.
	admin, err := e.groups.EnrollTelegramMember(ctx, chatID, adminID)
	if err != nil || admin.NewlyEnrolled {
		t.Fatalf("activator enroll = %+v, %v; want already enrolled", admin, err)
	}
}

func TestLeaveAndRejoin(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	act, err := e.groups.ActivateTelegramGroup(ctx, chatID, adminID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.groups.EnrollTelegramMember(ctx, chatID, aliceID); err != nil {
		t.Fatal(err)
	}

	changed, err := e.groups.RecordTelegramMemberLeft(ctx, chatID, aliceID)
	if err != nil || !changed {
		t.Fatalf("leave = %v, %v", changed, err)
	}
	if s := e.status(t, act.GroupID, aliceID); s != domain.MembershipStatusLeft {
		t.Fatalf("status = %q, want left", s)
	}

	res, err := e.groups.EnrollTelegramMember(ctx, chatID, aliceID)
	if err != nil || !res.NewlyEnrolled {
		t.Fatalf("rejoin = %+v, %v", res, err)
	}
	if s := e.status(t, act.GroupID, aliceID); s != domain.MembershipStatusActive {
		t.Fatalf("status = %q, want active", s)
	}
}

func TestRecordLeftIgnoresUnknownGroupsAndUsers(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	if changed, err := e.groups.RecordTelegramMemberLeft(ctx, chatID, aliceID); err != nil || changed {
		t.Fatalf("inactive group = %v, %v; want false, nil", changed, err)
	}
	if _, err := e.groups.ActivateTelegramGroup(ctx, chatID, adminID); err != nil {
		t.Fatal(err)
	}
	if changed, err := e.groups.RecordTelegramMemberLeft(ctx, chatID, aliceID); err != nil || changed {
		t.Fatalf("never-enrolled user = %v, %v; want false, nil", changed, err)
	}
	if _, err := e.users.FindTelegramUser(ctx, aliceID); !errors.Is(err, domain.ErrUserIdentityNotFound) {
		t.Fatal("leaving must not create a user")
	}
}

func TestMoveTelegramGroup(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	const newChatID int64 = -100999

	act, err := e.groups.ActivateTelegramGroup(ctx, chatID, adminID)
	if err != nil {
		t.Fatal(err)
	}
	moved, err := e.groups.MoveTelegramGroup(ctx, chatID, newChatID)
	if err != nil || !moved {
		t.Fatalf("move = %v, %v", moved, err)
	}
	// Telegram sends the announcement in both chats; the second is a no-op.
	if moved, err := e.groups.MoveTelegramGroup(ctx, chatID, newChatID); err != nil || moved {
		t.Fatalf("repeat move = %v, %v; want false, nil", moved, err)
	}

	res, err := e.groups.EnrollTelegramMember(ctx, newChatID, aliceID)
	if err != nil || res.GroupID != act.GroupID {
		t.Fatalf("enroll in new chat = %+v, %v; want group %q", res, err, act.GroupID)
	}
	if s := e.status(t, act.GroupID, adminID); s != domain.MembershipStatusActive {
		t.Fatalf("activator status after move = %q, want active", s)
	}
	if active, _ := e.groups.IsTelegramGroupActive(ctx, chatID); active {
		t.Fatal("old chat ID should no longer be active")
	}
}

func TestMoveTelegramGroupIgnoresInactiveChat(t *testing.T) {
	e := newEnv(t)
	if moved, err := e.groups.MoveTelegramGroup(context.Background(), chatID, -100999); err != nil || moved {
		t.Fatalf("move inactive = %v, %v; want false, nil", moved, err)
	}
}
