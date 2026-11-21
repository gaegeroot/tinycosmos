package telegram

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// StartGroupPayload is the deep-link parameter used in "add to group" links.
// Telegram passes it to the bot as "/start <payload>" in the group.
const StartGroupPayload = "activate"

// AddToGroupLink returns the deep link that lets a user pick a group, add the
// bot to it, and send /start there in one step.
func AddToGroupLink(botUsername string) string {
	return "https://t.me/" + url.PathEscape(botUsername) + "?startgroup=" + StartGroupPayload
}

// command is a parsed bot command such as "/join@TinyCosmosBot extra".
type command struct {
	name    string // lower case, without the slash
	payload string // the rest of the message after the command
}

// parseCommand extracts the bot command at the start of a message. It
// returns false if the message does not start with a command, or if the
// command is addressed to a different bot ("/join@OtherBot").
func parseCommand(m *models.Message, botUsername string) (command, bool) {
	if m == nil || !strings.HasPrefix(m.Text, "/") {
		return command{}, false
	}
	startsWithCommand := false
	for _, e := range m.Entities {
		if e.Type == models.MessageEntityTypeBotCommand && e.Offset == 0 {
			startsWithCommand = true
			break
		}
	}
	if !startsWithCommand {
		return command{}, false
	}

	head, rest, _ := strings.Cut(m.Text, " ")
	name, target, addressed := strings.Cut(strings.TrimPrefix(head, "/"), "@")
	if addressed && !strings.EqualFold(target, botUsername) {
		return command{}, false
	}
	if name == "" {
		return command{}, false
	}
	return command{name: strings.ToLower(name), payload: strings.TrimSpace(rest)}, true
}

// RegisterCommands publishes the bot's command menu to Telegram, so the
// commands appear when users type "/".
func RegisterCommands(ctx context.Context, b *bot.Bot) error {
	groupCommands := []models.BotCommand{
		{Command: "start", Description: "Set up Tiny Cosmos in this group (admins)"},
		{Command: "join", Description: "Take part in this group's weekly question"},
		{Command: "help", Description: "How Tiny Cosmos works"},
	}
	privateCommands := []models.BotCommand{
		{Command: "start", Description: "What Tiny Cosmos is and how to add it to a group"},
	}

	if _, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: groupCommands,
		Scope:    &models.BotCommandScopeAllGroupChats{},
	}); err != nil {
		return fmt.Errorf("telegram: set group commands: %w", err)
	}
	if _, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: privateCommands,
		Scope:    &models.BotCommandScopeAllPrivateChats{},
	}); err != nil {
		return fmt.Errorf("telegram: set private commands: %w", err)
	}
	return nil
}
