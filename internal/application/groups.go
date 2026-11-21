package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/gaegeroot/tinycosmos/internal/domain"
)

// TelegramGroupAdminVerifier asks Telegram whether a user administers a
// group chat. It is implemented by the Telegram adapter.
type TelegramGroupAdminVerifier interface {
	IsTelegramGroupAdmin(ctx context.Context, chatID, telegramUserID int64) (bool, error)
}

// GroupService activates Telegram groups and manages who takes part.
//
// Telegram bots cannot list a group's existing members, so membership is
// opt-in: the admin who activates the group is enrolled automatically, and
// everyone else enrolls themselves.
type GroupService struct {
	groups  GroupStore
	members MembershipStore
	users   *UserService
	admins  TelegramGroupAdminVerifier
	now     Clock
	newID   IDGenerator
}

// NewGroupService returns a GroupService. A nil clock or ID generator uses the
// system clock and random UUIDs.
func NewGroupService(
	groups GroupStore,
	members MembershipStore,
	users *UserService,
	admins TelegramGroupAdminVerifier,
	now Clock,
	newID IDGenerator,
) *GroupService {
	if now == nil {
		now = defaultClock
	}
	if newID == nil {
		newID = defaultIDGenerator
	}
	return &GroupService{groups: groups, members: members, users: users, admins: admins, now: now, newID: newID}
}

// ActivationResult describes the outcome of ActivateTelegramGroup.
type ActivationResult struct {
	GroupID domain.GroupID

	// NewlyActivated is false if the chat was already an active Tiny Cosmos
	// group. In that case nobody is enrolled and ActivatorUserID is empty.
	NewlyActivated bool

	// ActivatorUserID is the activating admin, who is enrolled automatically.
	ActivatorUserID domain.UserID
}

// ActivateTelegramGroup creates a Tiny Cosmos group for a Telegram chat and
// enrolls the activating admin.
//
// It is idempotent: if the chat is already active it returns the existing
// group with NewlyActivated false, without checking admin rights or enrolling
// anyone. Otherwise it returns ErrNotGroupAdmin unless Telegram confirms the
// requester administers the chat.
//
// The group and its Telegram integration are created atomically. Enrolling
// the activator happens afterwards; if that step fails the group stays
// active and the activator can enroll like anyone else.
func (s *GroupService) ActivateTelegramGroup(ctx context.Context, chatID, activatorTelegramUserID int64) (ActivationResult, error) {
	existing, err := s.findTelegramGroup(ctx, chatID)
	if err == nil {
		return ActivationResult{GroupID: existing}, nil
	}
	if !errors.Is(err, ErrGroupNotActivated) {
		return ActivationResult{}, err
	}

	isAdmin, err := s.admins.IsTelegramGroupAdmin(ctx, chatID, activatorTelegramUserID)
	if err != nil {
		return ActivationResult{}, fmt.Errorf("activate telegram group %d: verify admin: %w", chatID, err)
	}
	if !isAdmin {
		return ActivationResult{}, ErrNotGroupAdmin
	}

	group := domain.Group{ID: domain.GroupID(s.newID()), CreatedAt: s.now().UTC()}
	integration := domain.GroupIntegration{
		ID:              domain.GroupIntegrationID(s.newID()),
		GroupID:         group.ID,
		Provider:        domain.IntegrationProviderTelegram,
		ExternalGroupID: telegramExternalGroupID(chatID),
	}
	err = s.groups.CreateGroupWithIntegration(ctx, group, integration)
	if errors.Is(err, domain.ErrGroupIntegrationAlreadyExists) {
		// A concurrent activation of the same chat won.
		existing, findErr := s.findTelegramGroup(ctx, chatID)
		if findErr != nil {
			return ActivationResult{}, fmt.Errorf("activate telegram group %d after conflict: %w", chatID, findErr)
		}
		return ActivationResult{GroupID: existing}, nil
	}
	if err != nil {
		return ActivationResult{}, fmt.Errorf("activate telegram group %d: %w", chatID, err)
	}

	userID, err := s.enroll(ctx, group.ID, activatorTelegramUserID)
	if err != nil {
		return ActivationResult{GroupID: group.ID, NewlyActivated: true},
			fmt.Errorf("activate telegram group %d: enroll activator: %w", chatID, err)
	}
	return ActivationResult{GroupID: group.ID, NewlyActivated: true, ActivatorUserID: userID}, nil
}

// EnrollmentResult describes the outcome of EnrollTelegramMember.
type EnrollmentResult struct {
	GroupID domain.GroupID
	UserID  domain.UserID

	// NewlyEnrolled is false if the user was already an active member.
	NewlyEnrolled bool
}

// EnrollTelegramMember opts a Telegram user into an active group, creating
// their Tiny Cosmos user if needed. Members who previously left become
// active again. It returns ErrGroupNotActivated if the chat has not been
// activated.
func (s *GroupService) EnrollTelegramMember(ctx context.Context, chatID, telegramUserID int64) (EnrollmentResult, error) {
	groupID, err := s.findTelegramGroup(ctx, chatID)
	if err != nil {
		return EnrollmentResult{}, err
	}
	userID, _, err := s.users.ProvisionTelegramUser(ctx, telegramUserID)
	if err != nil {
		return EnrollmentResult{}, err
	}
	changed, err := s.members.ActivateMembership(ctx, groupID, userID)
	if err != nil {
		return EnrollmentResult{}, err
	}
	return EnrollmentResult{GroupID: groupID, UserID: userID, NewlyEnrolled: changed}, nil
}

// RecordTelegramMemberLeft marks a Telegram user's membership as left when
// they leave the chat. It reports whether a membership changed. Users who
// never enrolled, and chats that were never activated, are ignored; no user
// is created.
func (s *GroupService) RecordTelegramMemberLeft(ctx context.Context, chatID, telegramUserID int64) (bool, error) {
	groupID, err := s.findTelegramGroup(ctx, chatID)
	if errors.Is(err, ErrGroupNotActivated) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	userID, err := s.users.FindTelegramUser(ctx, telegramUserID)
	if errors.Is(err, domain.ErrUserIdentityNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return s.members.MarkMembershipLeft(ctx, groupID, userID)
}

// IsTelegramGroupActive reports whether a Telegram chat has been activated.
func (s *GroupService) IsTelegramGroupActive(ctx context.Context, chatID int64) (bool, error) {
	_, err := s.findTelegramGroup(ctx, chatID)
	if errors.Is(err, ErrGroupNotActivated) {
		return false, nil
	}
	return err == nil, err
}

// MoveTelegramGroup follows a Telegram chat that changed its ID, which
// happens when a basic group is upgraded to a supergroup. It reports whether
// an active group was moved; chats that were never activated are ignored.
// Telegram announces the move in both chats, so repeated calls are expected
// and harmless.
func (s *GroupService) MoveTelegramGroup(ctx context.Context, fromChatID, toChatID int64) (bool, error) {
	moved, err := s.groups.UpdateExternalGroupID(ctx, domain.IntegrationProviderTelegram,
		telegramExternalGroupID(fromChatID), telegramExternalGroupID(toChatID))
	if err != nil {
		return false, fmt.Errorf("move telegram group %d to %d: %w", fromChatID, toChatID, err)
	}
	return moved, nil
}

func (s *GroupService) enroll(ctx context.Context, groupID domain.GroupID, telegramUserID int64) (domain.UserID, error) {
	userID, _, err := s.users.ProvisionTelegramUser(ctx, telegramUserID)
	if err != nil {
		return "", err
	}
	if _, err := s.members.ActivateMembership(ctx, groupID, userID); err != nil {
		return "", err
	}
	return userID, nil
}

func (s *GroupService) findTelegramGroup(ctx context.Context, chatID int64) (domain.GroupID, error) {
	gi, err := s.groups.GetGroupIntegration(ctx, domain.IntegrationProviderTelegram, telegramExternalGroupID(chatID))
	if errors.Is(err, domain.ErrGroupIntegrationNotFound) {
		return "", ErrGroupNotActivated
	}
	if err != nil {
		return "", err
	}
	return gi.GroupID, nil
}

// telegramExternalGroupID is the canonical stored form of a Telegram chat ID.
func telegramExternalGroupID(chatID int64) string { return strconv.FormatInt(chatID, 10) }
