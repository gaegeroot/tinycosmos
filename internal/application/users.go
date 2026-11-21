package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/gaegeroot/tinycosmos/internal/domain"
)

// UserService provisions and looks up Tiny Cosmos users.
type UserService struct {
	users UserStore
	now   Clock
	newID IDGenerator
}

// NewUserService returns a UserService. A nil clock or ID generator uses the
// system clock and random UUIDs.
func NewUserService(users UserStore, now Clock, newID IDGenerator) *UserService {
	if now == nil {
		now = defaultClock
	}
	if newID == nil {
		newID = defaultIDGenerator
	}
	return &UserService{users: users, now: now, newID: newID}
}

// ProvisionTelegramUser returns the Tiny Cosmos user linked to a Telegram
// account, creating the user and its Telegram identity together if this is
// the account's first interaction. It reports whether a user was created.
//
// It is safe to call concurrently for the same account: if another request
// creates the user first, the existing user is returned.
func (s *UserService) ProvisionTelegramUser(ctx context.Context, telegramUserID int64) (domain.UserID, bool, error) {
	if telegramUserID <= 0 {
		return "", false, fmt.Errorf("provision telegram user: invalid telegram user id %d", telegramUserID)
	}

	if id, err := s.FindTelegramUser(ctx, telegramUserID); err == nil {
		return id, false, nil
	} else if !errors.Is(err, domain.ErrUserIdentityNotFound) {
		return "", false, err
	}

	now := s.now().UTC()
	user := domain.User{ID: domain.UserID(s.newID()), CreatedAt: now}
	identity := domain.UserIdentity{
		ID:             domain.UserIdentityID(s.newID()),
		UserID:         user.ID,
		Provider:       domain.IdentityProviderTelegram,
		ProviderUserID: telegramProviderUserID(telegramUserID),
		CreatedAt:      now,
	}

	err := s.users.CreateUserWithIdentity(ctx, user, identity)
	if errors.Is(err, domain.ErrUserIdentityAlreadyExists) {
		// A concurrent request linked this Telegram account first.
		id, findErr := s.FindTelegramUser(ctx, telegramUserID)
		if findErr != nil {
			return "", false, fmt.Errorf("provision telegram user %d after conflict: %w", telegramUserID, findErr)
		}
		return id, false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("provision telegram user %d: %w", telegramUserID, err)
	}
	return user.ID, true, nil
}

// FindTelegramUser returns the user linked to a Telegram account without
// creating one, or domain.ErrUserIdentityNotFound.
func (s *UserService) FindTelegramUser(ctx context.Context, telegramUserID int64) (domain.UserID, error) {
	identity, err := s.users.GetUserIdentityByProvider(ctx, domain.IdentityProviderTelegram, telegramProviderUserID(telegramUserID))
	if err != nil {
		return "", err
	}
	return identity.UserID, nil
}

// telegramProviderUserID is the canonical stored form of a Telegram user ID.
func telegramProviderUserID(id int64) string { return strconv.FormatInt(id, 10) }
