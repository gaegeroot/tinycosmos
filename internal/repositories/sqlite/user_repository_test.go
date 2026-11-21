package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaegeroot/tinycosmos/internal/domain"
)

var testTime = time.Date(2026, 11, 21, 12, 30, 45, 123_000_000, time.UTC)

func mustCreateUser(t *testing.T, r *UserRepository, id domain.UserID) domain.User {
	t.Helper()
	u := domain.User{ID: id, CreatedAt: testTime}
	if err := r.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("CreateUser(%q): %v", id, err)
	}
	return u
}

func telegramIdentity(id domain.UserIdentityID, userID domain.UserID, tgID string) domain.UserIdentity {
	return domain.UserIdentity{
		ID:             id,
		UserID:         userID,
		Provider:       domain.IdentityProviderTelegram,
		ProviderUserID: tgID,
		CreatedAt:      testTime,
	}
}

func TestOpenEnablesForeignKeys(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "fk.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Force several distinct pooled connections and check each one.
	db.SetMaxIdleConns(0)
	for i := 0; i < 3; i++ {
		var fk int
		if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
			t.Fatalf("PRAGMA foreign_keys: %v", err)
		}
		if fk != 1 {
			t.Fatalf("connection %d: foreign_keys = %d, want 1", i, fk)
		}
	}
}

func TestOpenRequiresPath(t *testing.T) {
	if _, err := Open(context.Background(), ""); err == nil {
		t.Fatal("Open(\"\") succeeded, want error")
	}
}

func TestCreateAndGetUser(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx := context.Background()

	want := mustCreateUser(t, r, "user-1")

	got, err := r.GetUserByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.ID != want.ID || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("GetUserByID = %+v, want %+v", got, want)
	}
	if got.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt location = %v, want UTC", got.CreatedAt.Location())
	}
}

func TestCreateUserStoresSchemaTimestampFormat(t *testing.T) {
	db := newTestDB(t)
	r := NewUserRepository(db)

	// A non-UTC input must be stored as UTC with millisecond precision.
	loc := time.FixedZone("UTC+2", 2*60*60)
	u := domain.User{ID: "user-1", CreatedAt: time.Date(2026, 11, 21, 14, 30, 45, 123_456_789, loc)}
	if err := r.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	var raw string
	if err := db.QueryRow(`SELECT created_at FROM users WHERE id = ?`, "user-1").Scan(&raw); err != nil {
		t.Fatalf("select created_at: %v", err)
	}
	if want := "2026-11-21T12:30:45.123Z"; raw != want {
		t.Fatalf("stored created_at = %q, want %q", raw, want)
	}
}

func TestGetUserReadsDatabaseDefaultTimestamp(t *testing.T) {
	db := newTestDB(t)
	r := NewUserRepository(db)

	// Rows created outside the repository use the migration's default format.
	if _, err := db.Exec(`INSERT INTO users (id) VALUES (?)`, "user-1"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := r.GetUserByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
}

func TestCreateUserDuplicateID(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	mustCreateUser(t, r, "user-1")

	err := r.CreateUser(context.Background(), domain.User{ID: "user-1", CreatedAt: testTime})
	if !errors.Is(err, domain.ErrUserAlreadyExists) {
		t.Fatalf("CreateUser duplicate = %v, want ErrUserAlreadyExists", err)
	}
}

func TestCreateUserInvalid(t *testing.T) {
	r := NewUserRepository(newTestDB(t))

	err := r.CreateUser(context.Background(), domain.User{ID: "user-1"})
	if !errors.Is(err, domain.ErrUserCreatedAtRequired) {
		t.Fatalf("CreateUser invalid = %v, want ErrUserCreatedAtRequired", err)
	}
}

func TestGetUserNotFound(t *testing.T) {
	r := NewUserRepository(newTestDB(t))

	_, err := r.GetUserByID(context.Background(), "missing")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("GetUserByID missing = %v, want ErrUserNotFound", err)
	}
}

func TestCreateAndGetUserIdentity(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx := context.Background()
	mustCreateUser(t, r, "user-1")

	want := telegramIdentity("ident-1", "user-1", "123456789")
	if err := r.CreateUserIdentity(ctx, want); err != nil {
		t.Fatalf("CreateUserIdentity: %v", err)
	}

	got, err := r.GetUserIdentityByProvider(ctx, domain.IdentityProviderTelegram, "123456789")
	if err != nil {
		t.Fatalf("GetUserIdentityByProvider: %v", err)
	}
	if got.ID != want.ID || got.UserID != want.UserID || got.Provider != want.Provider ||
		got.ProviderUserID != want.ProviderUserID || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("GetUserIdentityByProvider = %+v, want %+v", got, want)
	}
}

func TestUserCanHaveMultipleIdentities(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx := context.Background()
	mustCreateUser(t, r, "user-1")

	tg := telegramIdentity("ident-1", "user-1", "123")
	google := domain.UserIdentity{
		ID: "ident-2", UserID: "user-1", Provider: domain.IdentityProviderGoogle,
		ProviderUserID: "google-sub-abc", CreatedAt: testTime,
	}
	for _, id := range []domain.UserIdentity{tg, google} {
		if err := r.CreateUserIdentity(ctx, id); err != nil {
			t.Fatalf("CreateUserIdentity(%s): %v", id.Provider, err)
		}
	}
	for _, id := range []domain.UserIdentity{tg, google} {
		got, err := r.GetUserIdentityByProvider(ctx, id.Provider, id.ProviderUserID)
		if err != nil || got.UserID != "user-1" {
			t.Fatalf("GetUserIdentityByProvider(%s) = %+v, %v", id.Provider, got, err)
		}
	}
}

func TestSameProviderUserIDAcrossProvidersIsAllowed(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx := context.Background()
	mustCreateUser(t, r, "user-1")
	mustCreateUser(t, r, "user-2")

	if err := r.CreateUserIdentity(ctx, telegramIdentity("ident-1", "user-1", "42")); err != nil {
		t.Fatalf("telegram: %v", err)
	}
	phone := domain.UserIdentity{
		ID: "ident-2", UserID: "user-2", Provider: domain.IdentityProviderPhone,
		ProviderUserID: "42", CreatedAt: testTime,
	}
	if err := r.CreateUserIdentity(ctx, phone); err != nil {
		t.Fatalf("phone with same provider user id: %v", err)
	}
}

func TestCreateUserIdentityDuplicateExternalIdentity(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx := context.Background()
	mustCreateUser(t, r, "user-1")
	mustCreateUser(t, r, "user-2")

	if err := r.CreateUserIdentity(ctx, telegramIdentity("ident-1", "user-1", "123")); err != nil {
		t.Fatalf("first CreateUserIdentity: %v", err)
	}

	// Same Telegram account, different Tiny Cosmos user.
	err := r.CreateUserIdentity(ctx, telegramIdentity("ident-2", "user-2", "123"))
	if !errors.Is(err, domain.ErrUserIdentityAlreadyExists) {
		t.Fatalf("duplicate external identity = %v, want ErrUserIdentityAlreadyExists", err)
	}

	// The original link is unchanged.
	got, err := r.GetUserIdentityByProvider(ctx, domain.IdentityProviderTelegram, "123")
	if err != nil || got.UserID != "user-1" {
		t.Fatalf("after duplicate: got %+v, %v; want user-1", got, err)
	}
}

func TestCreateUserIdentityDuplicateID(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx := context.Background()
	mustCreateUser(t, r, "user-1")

	if err := r.CreateUserIdentity(ctx, telegramIdentity("ident-1", "user-1", "123")); err != nil {
		t.Fatalf("first CreateUserIdentity: %v", err)
	}
	err := r.CreateUserIdentity(ctx, telegramIdentity("ident-1", "user-1", "456"))
	if !errors.Is(err, domain.ErrUserIdentityAlreadyExists) {
		t.Fatalf("duplicate identity id = %v, want ErrUserIdentityAlreadyExists", err)
	}
}

func TestCreateUserIdentityUnknownUser(t *testing.T) {
	r := NewUserRepository(newTestDB(t))

	err := r.CreateUserIdentity(context.Background(), telegramIdentity("ident-1", "no-such-user", "123"))
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("CreateUserIdentity unknown user = %v, want ErrUserNotFound", err)
	}
}

func TestCreateUserIdentityInvalid(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	mustCreateUser(t, r, "user-1")

	id := telegramIdentity("ident-1", "user-1", "123")
	id.Provider = "facebook"
	err := r.CreateUserIdentity(context.Background(), id)
	if !errors.Is(err, domain.ErrUnknownIdentityProvider) {
		t.Fatalf("CreateUserIdentity invalid provider = %v, want ErrUnknownIdentityProvider", err)
	}
}

func TestGetUserIdentityNotFound(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx := context.Background()
	mustCreateUser(t, r, "user-1")
	if err := r.CreateUserIdentity(ctx, telegramIdentity("ident-1", "user-1", "123")); err != nil {
		t.Fatalf("CreateUserIdentity: %v", err)
	}

	cases := []struct {
		name     string
		provider domain.IdentityProvider
		id       string
	}{
		{"unknown provider user id", domain.IdentityProviderTelegram, "999"},
		{"same id under another provider", domain.IdentityProviderEmail, "123"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := r.GetUserIdentityByProvider(ctx, c.provider, c.id)
			if !errors.Is(err, domain.ErrUserIdentityNotFound) {
				t.Fatalf("got %v, want ErrUserIdentityNotFound", err)
			}
		})
	}
}

func TestGetUserIdentityUnsupportedProvider(t *testing.T) {
	r := NewUserRepository(newTestDB(t))

	_, err := r.GetUserIdentityByProvider(context.Background(), "facebook", "123")
	if !errors.Is(err, domain.ErrUnknownIdentityProvider) {
		t.Fatalf("got %v, want ErrUnknownIdentityProvider", err)
	}
}

func TestCanceledContext(t *testing.T) {
	r := NewUserRepository(newTestDB(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := r.CreateUser(ctx, domain.User{ID: "user-1", CreatedAt: testTime}); !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateUser with canceled ctx = %v, want context.Canceled", err)
	}
}
