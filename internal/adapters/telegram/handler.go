// Package telegram connects Tiny Cosmos to Telegram through
// github.com/go-telegram/bot. It turns Telegram updates into application
// calls and application results into messages; business rules stay in the
// application package.
package telegram

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gaegeroot/tinycosmos/internal/application"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Groups is the part of the application the handler uses. It is satisfied
// by *application.GroupService.
type Groups interface {
	ActivateTelegramGroup(ctx context.Context, chatID, telegramUserID int64) (application.ActivationResult, error)
	EnrollTelegramMember(ctx context.Context, chatID, telegramUserID int64) (application.EnrollmentResult, error)
	RecordTelegramMemberLeft(ctx context.Context, chatID, telegramUserID int64) (bool, error)
	IsTelegramGroupActive(ctx context.Context, chatID int64) (bool, error)
	MoveTelegramGroup(ctx context.Context, fromChatID, toChatID int64) (bool, error)
}

// DefaultBotAddedNudgeDelay is how long the bot waits after being added to a
// group before suggesting /start. Adding the bot through the deep link sends
// /start straight away, so the wait avoids a redundant hint in that case.
const DefaultBotAddedNudgeDelay = 5 * time.Second

// HandlerConfig configures a Handler.
type HandlerConfig struct {
	// BotID and BotUsername identify this bot, from getMe.
	BotID       int64
	BotUsername string

	// Logger receives operational logs. Nil uses slog.Default().
	Logger *slog.Logger

	// BotAddedNudgeDelay overrides DefaultBotAddedNudgeDelay when positive.
	BotAddedNudgeDelay time.Duration
}

// Handler processes Telegram updates. Its Handle method is a bot.HandlerFunc
// and is safe for concurrent use.
type Handler struct {
	groups      Groups
	botID       int64
	botUsername string
	log         *slog.Logger
	nudgeDelay  time.Duration
	pending     sync.WaitGroup
}

// NewHandler returns a Handler that acts on groups.
func NewHandler(groups Groups, cfg HandlerConfig) *Handler {
	h := &Handler{
		groups:      groups,
		botID:       cfg.BotID,
		botUsername: cfg.BotUsername,
		log:         cfg.Logger,
		nudgeDelay:  cfg.BotAddedNudgeDelay,
	}
	if h.log == nil {
		h.log = slog.Default()
	}
	if h.nudgeDelay <= 0 {
		h.nudgeDelay = DefaultBotAddedNudgeDelay
	}
	return h
}

// Wait blocks until background work started by Handle (delayed nudges) has
// finished. Cancel the context passed to Handle first so it ends promptly.
func (h *Handler) Wait() { h.pending.Wait() }

// Handle processes one update.
func (h *Handler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	switch {
	case update.Message != nil:
		h.handleMessage(ctx, b, update.Message)
	case update.CallbackQuery != nil:
		h.handleCallback(ctx, b, update.CallbackQuery)
	}
}

func (h *Handler) handleMessage(ctx context.Context, b *bot.Bot, m *models.Message) {
	switch m.Chat.Type {
	case models.ChatTypePrivate:
		h.handlePrivateMessage(ctx, b, m)
	case models.ChatTypeGroup, models.ChatTypeSupergroup:
		h.handleGroupMessage(ctx, b, m)
	}
}

func (h *Handler) handlePrivateMessage(ctx context.Context, b *bot.Bot, m *models.Message) {
	cmd, ok := parseCommand(m, h.botUsername)
	if !ok || (cmd.name != "start" && cmd.name != "help") {
		return
	}
	h.sendWithKeyboard(ctx, b, m.Chat.ID, msgPrivateWelcome, addToGroupKeyboard(AddToGroupLink(h.botUsername)))
}

func (h *Handler) handleGroupMessage(ctx context.Context, b *bot.Bot, m *models.Message) {
	chatID := m.Chat.ID

	// Service messages first: they never carry commands.
	switch {
	case m.MigrateToChatID != 0:
		h.moveGroup(ctx, chatID, m.MigrateToChatID)
		return
	case m.MigrateFromChatID != 0:
		h.moveGroup(ctx, m.MigrateFromChatID, chatID)
		return
	case len(m.NewChatMembers) > 0:
		h.onMembersAdded(ctx, b, chatID, m.NewChatMembers)
		return
	case m.LeftChatMember != nil:
		h.onMemberLeft(ctx, chatID, m.LeftChatMember)
		return
	}

	cmd, ok := parseCommand(m, h.botUsername)
	if !ok {
		return
	}
	switch cmd.name {
	case "start":
		h.activate(ctx, b, m)
	case "join":
		h.joinByCommand(ctx, b, m)
	case "help":
		h.help(ctx, b, chatID)
	}
}

// activate handles /start in a group, which is also what the
// ?startgroup= deep link sends.
func (h *Handler) activate(ctx context.Context, b *bot.Bot, m *models.Message) {
	chatID := m.Chat.ID
	if isAnonymous(m) {
		h.send(ctx, b, chatID, msgAnonymous)
		return
	}
	if m.From.IsBot {
		return
	}

	res, err := h.groups.ActivateTelegramGroup(ctx, chatID, m.From.ID)
	switch {
	case errors.Is(err, application.ErrNotGroupAdmin):
		h.send(ctx, b, chatID, msgNotAdmin)
		return
	case err != nil && !res.NewlyActivated:
		h.log.Error("activate group", "chat_id", chatID, "user_id", m.From.ID, "err", err)
		h.send(ctx, b, chatID, msgSomethingWentWrong)
		return
	case err != nil:
		// The group is active but enrolling the activator failed. Onboard
		// anyway; the activator can tap "I'm in" like everyone else.
		h.log.Error("enroll activator", "chat_id", chatID, "user_id", m.From.ID, "err", err)
	case !res.NewlyActivated:
		h.sendWithKeyboard(ctx, b, chatID, msgAlreadyActive, joinKeyboard())
		return
	}

	h.log.Info("group activated", "chat_id", chatID, "group_id", res.GroupID, "activator_user_id", res.ActivatorUserID)
	activator := ""
	if res.ActivatorUserID != "" {
		activator = displayName(m.From)
	}
	// Stop the sequence at the first failure so messages never arrive out
	// of order or with gaps.
	if !h.send(ctx, b, chatID, msgOnboardingWelcome) {
		return
	}
	if !h.send(ctx, b, chatID, msgHowItWorks) {
		return
	}
	h.sendWithKeyboard(ctx, b, chatID, msgJoinInvite(activator), joinKeyboard())
}

func (h *Handler) joinByCommand(ctx context.Context, b *bot.Bot, m *models.Message) {
	chatID := m.Chat.ID
	if isAnonymous(m) {
		h.send(ctx, b, chatID, msgAnonymous)
		return
	}
	if m.From.IsBot {
		return
	}

	res, err := h.groups.EnrollTelegramMember(ctx, chatID, m.From.ID)
	switch {
	case errors.Is(err, application.ErrGroupNotActivated):
		h.send(ctx, b, chatID, msgNotActivated)
	case err != nil:
		h.log.Error("enroll member", "chat_id", chatID, "user_id", m.From.ID, "err", err)
		h.send(ctx, b, chatID, msgSomethingWentWrong)
	case res.NewlyEnrolled:
		h.log.Info("member enrolled", "chat_id", chatID, "group_id", res.GroupID, "user_id", res.UserID)
		h.send(ctx, b, chatID, msgEnrolled(displayName(m.From)))
	default:
		h.send(ctx, b, chatID, msgAlreadyEnrolled(displayName(m.From)))
	}
}

func (h *Handler) handleCallback(ctx context.Context, b *bot.Bot, cq *models.CallbackQuery) {
	toast := ""
	defer func() {
		// Always answer, or the button keeps showing a loading spinner.
		if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: cq.ID,
			Text:            toast,
		}); err != nil {
			h.log.Warn("answer callback query", "err", err)
		}
	}()

	if cq.Data != joinCallbackData {
		return
	}
	chat := callbackChat(cq)
	if chat == nil || (chat.Type != models.ChatTypeGroup && chat.Type != models.ChatTypeSupergroup) {
		return
	}

	res, err := h.groups.EnrollTelegramMember(ctx, chat.ID, cq.From.ID)
	switch {
	case errors.Is(err, application.ErrGroupNotActivated):
		toast = toastNotActivated
	case err != nil:
		h.log.Error("enroll member", "chat_id", chat.ID, "user_id", cq.From.ID, "err", err)
		toast = toastError
	case res.NewlyEnrolled:
		h.log.Info("member enrolled", "chat_id", chat.ID, "group_id", res.GroupID, "user_id", res.UserID)
		toast = toastJoined
		h.send(ctx, b, chat.ID, msgEnrolled(displayName(&cq.From)))
	default:
		toast = toastAlreadyJoined
	}
}

func (h *Handler) help(ctx context.Context, b *bot.Bot, chatID int64) {
	active, err := h.groups.IsTelegramGroupActive(ctx, chatID)
	if err != nil {
		h.log.Error("check group active", "chat_id", chatID, "err", err)
		h.send(ctx, b, chatID, msgSomethingWentWrong)
		return
	}
	if !active {
		h.send(ctx, b, chatID, msgHowItWorks+"\n\n"+msgNotActivated)
		return
	}
	h.sendWithKeyboard(ctx, b, chatID, msgHowItWorks, joinKeyboard())
}

func (h *Handler) onMembersAdded(ctx context.Context, b *bot.Bot, chatID int64, added []models.User) {
	botAdded := false
	var names []string
	for i := range added {
		switch {
		case added[i].ID == h.botID:
			botAdded = true
		case !added[i].IsBot:
			names = append(names, displayName(&added[i]))
		}
	}

	if botAdded {
		h.nudgeIfNotActivated(ctx, b, chatID)
	}
	if len(names) == 0 {
		return
	}
	// New arrivals are invited, never enrolled automatically.
	active, err := h.groups.IsTelegramGroupActive(ctx, chatID)
	if err != nil {
		h.log.Error("check group active", "chat_id", chatID, "err", err)
		return
	}
	if active {
		h.sendWithKeyboard(ctx, b, chatID, msgMembersJoinedChat(names), joinKeyboard())
	}
}

// nudgeIfNotActivated suggests /start shortly after the bot is added, unless
// the group was activated in the meantime (as it is with the deep link).
func (h *Handler) nudgeIfNotActivated(ctx context.Context, b *bot.Bot, chatID int64) {
	h.pending.Add(1)
	go func() {
		defer h.pending.Done()
		t := time.NewTimer(h.nudgeDelay)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		active, err := h.groups.IsTelegramGroupActive(ctx, chatID)
		if err != nil {
			h.log.Error("check group active", "chat_id", chatID, "err", err)
			return
		}
		if !active {
			h.send(ctx, b, chatID, msgBotAdded)
		}
	}()
}

func (h *Handler) onMemberLeft(ctx context.Context, chatID int64, u *models.User) {
	if u.ID == h.botID {
		// The bot was removed. Group data is kept; re-adding it resumes.
		h.log.Info("bot removed from group", "chat_id", chatID)
		return
	}
	if u.IsBot {
		return
	}
	changed, err := h.groups.RecordTelegramMemberLeft(ctx, chatID, u.ID)
	if err != nil {
		h.log.Error("record member left", "chat_id", chatID, "user_id", u.ID, "err", err)
		return
	}
	if changed {
		h.log.Info("member left", "chat_id", chatID, "user_id", u.ID)
	}
}

func (h *Handler) moveGroup(ctx context.Context, fromChatID, toChatID int64) {
	moved, err := h.groups.MoveTelegramGroup(ctx, fromChatID, toChatID)
	if err != nil {
		h.log.Error("move group", "from_chat_id", fromChatID, "to_chat_id", toChatID, "err", err)
		return
	}
	if moved {
		h.log.Info("group moved to new chat id", "from_chat_id", fromChatID, "to_chat_id", toChatID)
	}
}

// send posts an HTML message and reports whether it was delivered.
func (h *Handler) send(ctx context.Context, b *bot.Bot, chatID int64, text string) bool {
	return h.sendParams(ctx, b, &bot.SendMessageParams{ChatID: chatID, Text: text, ParseMode: models.ParseModeHTML})
}

func (h *Handler) sendWithKeyboard(ctx context.Context, b *bot.Bot, chatID int64, text string, kb models.InlineKeyboardMarkup) bool {
	return h.sendParams(ctx, b, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: kb,
	})
}

func (h *Handler) sendParams(ctx context.Context, b *bot.Bot, p *bot.SendMessageParams) bool {
	if _, err := b.SendMessage(ctx, p); err != nil {
		h.log.Error("send message", "chat_id", p.ChatID, "err", err)
		return false
	}
	return true
}

// isAnonymous reports whether a group message hides its sender: anonymous
// admins and channels post as a chat, so the person can't be identified.
func isAnonymous(m *models.Message) bool {
	return m.SenderChat != nil || m.From == nil
}

// callbackChat returns the chat of the message a button was attached to.
func callbackChat(cq *models.CallbackQuery) *models.Chat {
	switch {
	case cq.Message.Message != nil:
		return &cq.Message.Message.Chat
	case cq.Message.InaccessibleMessage != nil:
		return &cq.Message.InaccessibleMessage.Chat
	}
	return nil
}
