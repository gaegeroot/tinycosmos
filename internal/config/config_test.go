package config

import (
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadBot(t *testing.T) {
	cfg, err := LoadBot(env(map[string]string{EnvTelegramBotToken: " 123:abc \n"}))
	if err != nil {
		t.Fatalf("LoadBot: %v", err)
	}
	if cfg.TelegramBotToken != "123:abc" || cfg.DatabasePath != DefaultDatabasePath {
		t.Fatalf("cfg = %+v", cfg)
	}

	cfg, err = LoadBot(env(map[string]string{EnvTelegramBotToken: "t", EnvDatabasePath: "/tmp/x.db"}))
	if err != nil || cfg.DatabasePath != "/tmp/x.db" {
		t.Fatalf("custom path: %+v, %v", cfg, err)
	}
}

func TestLoadBotRequiresToken(t *testing.T) {
	_, err := LoadBot(env(nil))
	if err == nil || !strings.Contains(err.Error(), EnvTelegramBotToken) {
		t.Fatalf("err = %v, want missing-token error", err)
	}
}
