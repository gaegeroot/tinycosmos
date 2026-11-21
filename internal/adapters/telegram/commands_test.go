package telegram

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestParseCommand(t *testing.T) {
	cmdEntity := func(n int) []models.MessageEntity {
		return []models.MessageEntity{{Type: models.MessageEntityTypeBotCommand, Offset: 0, Length: n}}
	}
	tests := []struct {
		name     string
		msg      *models.Message
		wantOK   bool
		wantName string
		wantArgs string
	}{
		{"plain", &models.Message{Text: "/join", Entities: cmdEntity(5)}, true, "join", ""},
		{"addressed to us", &models.Message{Text: "/start@TinyCosmosBot activate", Entities: cmdEntity(20)}, true, "start", "activate"},
		{"addressed case-insensitively", &models.Message{Text: "/start@tinycosmosbot", Entities: cmdEntity(20)}, true, "start", ""},
		{"upper-case command", &models.Message{Text: "/HELP", Entities: cmdEntity(5)}, true, "help", ""},
		{"other bot", &models.Message{Text: "/join@OtherBot", Entities: cmdEntity(14)}, false, "", ""},
		{"no entity", &models.Message{Text: "/join"}, false, "", ""},
		{"entity not at start", &models.Message{Text: "hi /join", Entities: []models.MessageEntity{{Type: models.MessageEntityTypeBotCommand, Offset: 3, Length: 5}}}, false, "", ""},
		{"plain text", &models.Message{Text: "hello"}, false, "", ""},
		{"bare slash", &models.Message{Text: "/", Entities: cmdEntity(1)}, false, "", ""},
		{"nil", nil, false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseCommand(tt.msg, "TinyCosmosBot")
			if ok != tt.wantOK || got.name != tt.wantName || got.payload != tt.wantArgs {
				t.Fatalf("parseCommand = %+v, %v; want {%s %s}, %v", got, ok, tt.wantName, tt.wantArgs, tt.wantOK)
			}
		})
	}
}

func TestAddToGroupLink(t *testing.T) {
	if got, want := AddToGroupLink("TinyCosmosBot"), "https://t.me/TinyCosmosBot?startgroup=activate"; got != want {
		t.Fatalf("AddToGroupLink = %q, want %q", got, want)
	}
}

func TestJoinNames(t *testing.T) {
	for _, tt := range []struct {
		in   []string
		want string
	}{
		{nil, ""},
		{[]string{"A"}, "A"},
		{[]string{"A", "B"}, "A and B"},
		{[]string{"A", "B", "C"}, "A, B and C"},
	} {
		if got := joinNames(tt.in); got != tt.want {
			t.Errorf("joinNames(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
