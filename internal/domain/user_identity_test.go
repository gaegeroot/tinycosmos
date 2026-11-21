package domain

import (
	"errors"
	"testing"
	"time"
)

func TestIdentityProviderValid(t *testing.T) {
	for _, p := range IdentityProviders() {
		if !p.Valid() {
			t.Errorf("%q.Valid() = false, want true", p)
		}
	}
	for _, p := range []IdentityProvider{"", "Telegram", "TELEGRAM", " telegram", "facebook", "sms"} {
		if p.Valid() {
			t.Errorf("%q.Valid() = true, want false", p)
		}
	}
}

func TestIdentityProvidersMatchSchema(t *testing.T) {
	// Must stay in sync with the user_identities.provider CHECK constraint in
	// database/migrations/001_initial_schema.sql.
	want := []string{"telegram", "email", "apple", "google", "phone"}
	got := IdentityProviders()
	if len(got) != len(want) {
		t.Fatalf("IdentityProviders() has %d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].String() != w {
			t.Errorf("IdentityProviders()[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestIdentityProvidersReturnsCopy(t *testing.T) {
	ps := IdentityProviders()
	ps[0] = "mutated"
	if IdentityProviders()[0] != IdentityProviderTelegram {
		t.Fatal("IdentityProviders() result shares state across calls")
	}
}

func TestParseIdentityProvider(t *testing.T) {
	for _, p := range IdentityProviders() {
		got, err := ParseIdentityProvider(string(p))
		if err != nil || got != p {
			t.Errorf("ParseIdentityProvider(%q) = %q, %v; want %q, nil", p, got, err, p)
		}
	}
	for _, s := range []string{"", "Google", "github"} {
		got, err := ParseIdentityProvider(s)
		if !errors.Is(err, ErrUnknownIdentityProvider) || got != "" {
			t.Errorf("ParseIdentityProvider(%q) = %q, %v; want \"\", ErrUnknownIdentityProvider", s, got, err)
		}
	}
}

func TestUserIdentityValidate(t *testing.T) {
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	valid := UserIdentity{
		ID:             "i-1",
		UserID:         "u-1",
		Provider:       IdentityProviderTelegram,
		ProviderUserID: "123456789",
		CreatedAt:      now,
	}

	tests := []struct {
		name   string
		mutate func(*UserIdentity)
		want   error
	}{
		{"valid", func(*UserIdentity) {}, nil},
		{"missing id", func(i *UserIdentity) { i.ID = "" }, ErrUserIdentityIDRequired},
		{"missing user id", func(i *UserIdentity) { i.UserID = "" }, ErrUserIdentityUserIDRequired},
		{"missing provider", func(i *UserIdentity) { i.Provider = "" }, ErrUnknownIdentityProvider},
		{"unknown provider", func(i *UserIdentity) { i.Provider = "facebook" }, ErrUnknownIdentityProvider},
		{"missing provider user id", func(i *UserIdentity) { i.ProviderUserID = "" }, ErrProviderUserIDRequired},
		{"blank provider user id", func(i *UserIdentity) { i.ProviderUserID = " \t" }, ErrProviderUserIDRequired},
		{"missing created_at", func(i *UserIdentity) { i.CreatedAt = time.Time{} }, ErrUserIdentityCreatedAtRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := valid
			tt.mutate(&id)
			if got := id.Validate(); !errors.Is(got, tt.want) {
				t.Fatalf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
