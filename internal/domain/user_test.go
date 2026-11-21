package domain

import (
	"errors"
	"testing"
	"time"
)

func TestUserValidate(t *testing.T) {
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		user User
		want error
	}{
		{"valid", User{ID: "u-1", CreatedAt: now}, nil},
		{"missing id", User{CreatedAt: now}, ErrUserIDRequired},
		{"blank id", User{ID: "  ", CreatedAt: now}, ErrUserIDRequired},
		{"missing created_at", User{ID: "u-1"}, ErrUserCreatedAtRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.Validate(); !errors.Is(got, tt.want) {
				t.Fatalf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
