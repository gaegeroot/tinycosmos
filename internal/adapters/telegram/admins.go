package telegram

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// AdminVerifier checks group admin rights with Telegram. It implements
// application.TelegramGroupAdminVerifier.
type AdminVerifier struct {
	api *bot.Bot
}

// NewAdminVerifier returns an AdminVerifier that queries Telegram through api.
func NewAdminVerifier(api *bot.Bot) *AdminVerifier {
	return &AdminVerifier{api: api}
}

// IsTelegramGroupAdmin reports whether the user is the owner or an
// administrator of the chat, according to Telegram right now.
func (v *AdminVerifier) IsTelegramGroupAdmin(ctx context.Context, chatID, telegramUserID int64) (bool, error) {
	member, err := v.api.GetChatMember(ctx, &bot.GetChatMemberParams{ChatID: chatID, UserID: telegramUserID})
	if err != nil {
		return false, fmt.Errorf("telegram: get chat member: %w", err)
	}
	switch member.Type {
	case models.ChatMemberTypeOwner, models.ChatMemberTypeAdministrator:
		return true, nil
	}
	return false, nil
}
