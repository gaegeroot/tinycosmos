// Package config loads runtime configuration from environment variables.
package config

import (
	"errors"
	"strings"
)

// Environment variable names.
const (
	EnvTelegramBotToken = "TELEGRAM_BOT_TOKEN"
	EnvDatabasePath     = "TINY_COSMOS_DB_PATH"
)

// DefaultDatabasePath is used when EnvDatabasePath is unset. It is relative
// to the working directory, so run binaries from the repository root.
const DefaultDatabasePath = "data/tiny-cosmos.db"

// Bot is the configuration for cmd/bot.
type Bot struct {
	TelegramBotToken string
	DatabasePath     string
}

// LoadBot reads the bot configuration using getenv (normally os.Getenv).
func LoadBot(getenv func(string) string) (Bot, error) {
	cfg := Bot{
		TelegramBotToken: strings.TrimSpace(getenv(EnvTelegramBotToken)),
		DatabasePath:     strings.TrimSpace(getenv(EnvDatabasePath)),
	}
	if cfg.TelegramBotToken == "" {
		return Bot{}, errors.New("config: " + EnvTelegramBotToken + " is not set. Put it in .env in the " +
			"directory you run the bot from (see .env.example), or export it in your shell")
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = DefaultDatabasePath
	}
	return cfg, nil
}
