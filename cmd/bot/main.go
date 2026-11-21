// Command bot runs the Tiny Cosmos Telegram bot using long polling.
//
// Configuration comes from the environment (see internal/config):
//
//	TELEGRAM_BOT_TOKEN   required, from @BotFather
//	TINY_COSMOS_DB_PATH  optional, defaults to data/tiny-cosmos.db
//
// The database is created and migrated on startup.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gaegeroot/tinycosmos/database"
	"github.com/gaegeroot/tinycosmos/internal/adapters/telegram"
	"github.com/gaegeroot/tinycosmos/internal/application"
	"github.com/gaegeroot/tinycosmos/internal/config"
	"github.com/gaegeroot/tinycosmos/internal/repositories/sqlite"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("bot stopped with an error", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadBot(os.Getenv)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}
	db, err := sqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()
	version, err := sqlite.Migrate(ctx, db, database.Migrations())
	if err != nil {
		return err
	}

	// The handler is created after the client because admin checks need the
	// client. No updates arrive before Start, so this ordering is safe.
	var handler *telegram.Handler
	api, err := bot.New(cfg.TelegramBotToken,
		bot.WithSkipGetMe(),
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, u *models.Update) {
			handler.Handle(ctx, b, u)
		}),
		bot.WithAllowedUpdates(bot.AllowedUpdates{models.AllowedUpdateMessage, models.AllowedUpdateCallbackQuery}),
		bot.WithErrorsHandler(func(err error) { logger.Error("telegram", "err", err) }),
	)
	if err != nil {
		return fmt.Errorf("create telegram client: %w", err)
	}
	me, err := api.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("connect to telegram (check %s): %w", config.EnvTelegramBotToken, err)
	}

	users := application.NewUserService(sqlite.NewUserRepository(db), nil, nil)
	groups := application.NewGroupService(
		sqlite.NewGroupRepository(db),
		sqlite.NewMembershipRepository(db),
		users,
		telegram.NewAdminVerifier(api),
		nil, nil,
	)
	handler = telegram.NewHandler(groups, telegram.HandlerConfig{
		BotID:       me.ID,
		BotUsername: me.Username,
		Logger:      logger,
	})

	if err := telegram.RegisterCommands(ctx, api); err != nil {
		logger.Warn("could not register the command menu; commands still work", "err", err)
	}

	logger.Info("bot running; press Ctrl+C to stop",
		"username", "@"+me.Username,
		"add_to_group_link", telegram.AddToGroupLink(me.Username),
		"database", cfg.DatabasePath,
		"schema_version", version,
	)
	api.Start(ctx) // blocks until ctx is cancelled and running handlers finish
	handler.Wait()
	logger.Info("bot stopped")
	return nil
}
