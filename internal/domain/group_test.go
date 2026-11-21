package domain

import (
	"errors"
	"testing"
	"time"
)

func TestGroupValidate(t *testing.T) {
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		group Group
		want  error
	}{
		{"valid", Group{ID: "g-1", CreatedAt: now}, nil},
		{"missing id", Group{CreatedAt: now}, ErrGroupIDRequired},
		{"blank id", Group{ID: " ", CreatedAt: now}, ErrGroupIDRequired},
		{"missing created_at", Group{ID: "g-1"}, ErrGroupCreatedAtRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.group.Validate(); !errors.Is(got, tt.want) {
				t.Fatalf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntegrationProvider(t *testing.T) {
	if !IntegrationProviderTelegram.Valid() {
		t.Fatal("telegram should be valid")
	}
	for _, s := range []string{"", "Telegram", "slack"} {
		if _, err := ParseIntegrationProvider(s); !errors.Is(err, ErrUnknownIntegrationProvider) {
			t.Errorf("ParseIntegrationProvider(%q) err = %v, want ErrUnknownIntegrationProvider", s, err)
		}
	}
	if p, err := ParseIntegrationProvider("telegram"); err != nil || p != IntegrationProviderTelegram {
		t.Fatalf("ParseIntegrationProvider(telegram) = %q, %v", p, err)
	}
}

func TestGroupIntegrationValidate(t *testing.T) {
	valid := GroupIntegration{ID: "gi-1", GroupID: "g-1", Provider: IntegrationProviderTelegram, ExternalGroupID: "-100123"}
	tests := []struct {
		name   string
		mutate func(*GroupIntegration)
		want   error
	}{
		{"valid", func(*GroupIntegration) {}, nil},
		{"missing id", func(i *GroupIntegration) { i.ID = "" }, ErrGroupIntegrationIDRequired},
		{"missing group id", func(i *GroupIntegration) { i.GroupID = "" }, ErrGroupIntegrationGroupIDRequired},
		{"unknown provider", func(i *GroupIntegration) { i.Provider = "slack" }, ErrUnknownIntegrationProvider},
		{"missing external id", func(i *GroupIntegration) { i.ExternalGroupID = " " }, ErrExternalGroupIDRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gi := valid
			tt.mutate(&gi)
			if got := gi.Validate(); !errors.Is(got, tt.want) {
				t.Fatalf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMembershipStatus(t *testing.T) {
	for _, s := range []MembershipStatus{MembershipStatusActive, MembershipStatusLeft} {
		if got, err := ParseMembershipStatus(string(s)); err != nil || got != s {
			t.Errorf("ParseMembershipStatus(%q) = %q, %v", s, got, err)
		}
	}
	for _, s := range []string{"", "Active", "banned"} {
		if _, err := ParseMembershipStatus(s); !errors.Is(err, ErrUnknownMembershipStatus) {
			t.Errorf("ParseMembershipStatus(%q) err = %v, want ErrUnknownMembershipStatus", s, err)
		}
	}
}
