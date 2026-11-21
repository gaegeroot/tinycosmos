package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaegeroot/tinycosmos/database"
	"github.com/gaegeroot/tinycosmos/internal/application"
	"github.com/gaegeroot/tinycosmos/internal/domain"
	"github.com/gaegeroot/tinycosmos/internal/repositories/sqlite"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	botID       int64 = 999
	botUsername       = "TinyCosmosBot"
	groupChatID int64 = -100500
	adminTGID   int64 = 1001
	aliceTGID   int64 = 1002
)

type harness struct {
	t       *testing.T
	tg      *fakeTelegram
	api     *bot.Bot
	h       *Handler
	groups  *application.GroupService
	users   *application.UserService
	members *sqlite.MembershipRepository
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctx := context.Background()

	db, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := sqlite.Migrate(ctx, db, database.Migrations()); err != nil {
		t.Fatal(err)
	}

	tg := newFakeTelegram(t)
	tg.memberTypes[strconv.FormatInt(adminTGID, 10)] = "creator"
	api := tg.newBot()

	var n atomic.Int64
	newID := func() string { return fmt.Sprintf("id-%d", n.Add(1)) }
	members := sqlite.NewMembershipRepository(db)
	users := application.NewUserService(sqlite.NewUserRepository(db), nil, newID)
	groups := application.NewGroupService(sqlite.NewGroupRepository(db), members, users, NewAdminVerifier(api), nil, newID)

	h := NewHandler(groups, HandlerConfig{
		BotID:              botID,
		BotUsername:        botUsername,
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		BotAddedNudgeDelay: 20 * time.Millisecond,
	})
	return &harness{t: t, tg: tg, api: api, h: h, groups: groups, users: users, members: members}
}

func (hs *harness) handle(u *models.Update) {
	hs.t.Helper()
	hs.h.Handle(context.Background(), hs.api, u)
}

func (hs *harness) activate() {
	hs.t.Helper()
	hs.handle(commandUpdate(groupChatID, adminUser(), "/start@"+botUsername+" "+StartGroupPayload))
	if active, err := hs.groups.IsTelegramGroupActive(context.Background(), groupChatID); err != nil || !active {
		hs.t.Fatalf("group not active after activation: %v, %v", active, err)
	}
	hs.tg.reset()
}

func (hs *harness) membership(telegramUserID int64) (domain.MembershipStatus, error) {
	ctx := context.Background()
	uid, err := hs.users.FindTelegramUser(ctx, telegramUserID)
	if err != nil {
		return "", err
	}
	res, err := hs.groups.ActivateTelegramGroup(ctx, groupChatID, adminTGID) // returns existing group
	if err != nil {
		return "", err
	}
	return hs.members.GetMembershipStatus(ctx, res.GroupID, uid)
}

func adminUser() *models.User {
	return &models.User{ID: adminTGID, FirstName: "Ada"}
}

func aliceUser() *models.User {
	return &models.User{ID: aliceTGID, FirstName: "Alice"}
}

func groupChat(id int64) models.Chat {
	return models.Chat{ID: id, Type: models.ChatTypeGroup, Title: "Friends"}
}

func commandUpdate(chatID int64, from *models.User, text string) *models.Update {
	head, _, _ := strings.Cut(text, " ")
	chat := groupChat(chatID)
	if chatID > 0 {
		chat = models.Chat{ID: chatID, Type: models.ChatTypePrivate}
	}
	return &models.Update{Message: &models.Message{
		ID:       1,
		Chat:     chat,
		From:     from,
		Text:     text,
		Entities: []models.MessageEntity{{Type: models.MessageEntityTypeBotCommand, Offset: 0, Length: len(head)}},
	}}
}

func callbackUpdate(from models.User, data string) *models.Update {
	return &models.Update{CallbackQuery: &models.CallbackQuery{
		ID:      "cq-1",
		From:    from,
		Data:    data,
		Message: models.MaybeInaccessibleMessage{Type: models.MaybeInaccessibleMessageTypeMessage, Message: &models.Message{ID: 7, Chat: groupChat(groupChatID)}},
	}}
}

func keyboardOf(t *testing.T, c apiCall) models.InlineKeyboardMarkup {
	t.Helper()
	var kb models.InlineKeyboardMarkup
	if err := json.Unmarshal([]byte(c.fields["reply_markup"]), &kb); err != nil {
		t.Fatalf("reply_markup %q: %v", c.fields["reply_markup"], err)
	}
	return kb
}

func wantTexts(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("sent %d messages %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("message %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDeepLinkActivationByAdminSendsOnboarding(t *testing.T) {
	hs := newHarness(t)

	hs.handle(commandUpdate(groupChatID, adminUser(), "/start@"+botUsername+" "+StartGroupPayload))

	wantTexts(t, hs.tg.sentTexts(), msgOnboardingWelcome, msgHowItWorks, msgJoinInvite("Ada"))
	sends := hs.tg.callsTo("sendMessage")
	for _, c := range sends {
		if c.fields["chat_id"] != strconv.FormatInt(groupChatID, 10) || c.fields["parse_mode"] != "HTML" {
			t.Fatalf("send fields = %v", c.fields)
		}
	}
	kb := keyboardOf(t, sends[2])
	if len(kb.InlineKeyboard) != 1 || kb.InlineKeyboard[0][0].CallbackData != joinCallbackData {
		t.Fatalf("join keyboard = %+v", kb)
	}

	// Telegram was asked whether the sender is an admin.
	gcm := hs.tg.callsTo("getChatMember")
	if len(gcm) != 1 || gcm[0].fields["user_id"] != strconv.FormatInt(adminTGID, 10) {
		t.Fatalf("getChatMember calls = %+v", gcm)
	}
	if s, err := hs.membership(adminTGID); err != nil || s != domain.MembershipStatusActive {
		t.Fatalf("activator membership = %q, %v; want active", s, err)
	}
}

func TestPlainStartWithoutPayloadAlsoActivates(t *testing.T) {
	hs := newHarness(t)
	hs.handle(commandUpdate(groupChatID, adminUser(), "/start"))
	if len(hs.tg.sentTexts()) != 3 {
		t.Fatalf("sent %q, want onboarding sequence", hs.tg.sentTexts())
	}
}

func TestStartByNonAdmin(t *testing.T) {
	hs := newHarness(t)

	hs.handle(commandUpdate(groupChatID, aliceUser(), "/start"))

	wantTexts(t, hs.tg.sentTexts(), msgNotAdmin)
	if active, _ := hs.groups.IsTelegramGroupActive(context.Background(), groupChatID); active {
		t.Fatal("non-admin activated the group")
	}
}

func TestStartWhenAlreadyActive(t *testing.T) {
	hs := newHarness(t)
	hs.activate()

	hs.handle(commandUpdate(groupChatID, aliceUser(), "/start"))

	wantTexts(t, hs.tg.sentTexts(), msgAlreadyActive)
	if kb := keyboardOf(t, hs.tg.callsTo("sendMessage")[0]); kb.InlineKeyboard[0][0].CallbackData != joinCallbackData {
		t.Fatalf("keyboard = %+v", kb)
	}
	if _, err := hs.membership(aliceTGID); !errors.Is(err, domain.ErrUserIdentityNotFound) {
		t.Fatalf("re-sending /start enrolled the sender: %v", err)
	}
}

func TestStartFromAnonymousAdmin(t *testing.T) {
	hs := newHarness(t)
	u := commandUpdate(groupChatID, &models.User{ID: 1087968824, IsBot: true, FirstName: "Group"}, "/start")
	chat := groupChat(groupChatID)
	u.Message.SenderChat = &chat

	hs.handle(u)

	wantTexts(t, hs.tg.sentTexts(), msgAnonymous)
	if len(hs.tg.callsTo("getChatMember")) != 0 {
		t.Fatal("admin check ran for an anonymous sender")
	}
}

func TestOnboardingStopsAtFirstFailedMessage(t *testing.T) {
	hs := newHarness(t)
	hs.tg.sendsBefore = 1 // the second message fails

	hs.handle(commandUpdate(groupChatID, adminUser(), "/start"))

	if got := len(hs.tg.callsTo("sendMessage")); got != 2 {
		t.Fatalf("sendMessage attempts = %d, want 2 (stop after failure)", got)
	}
}

func TestAdminCheckFailureReportsError(t *testing.T) {
	hs := newHarness(t)
	hs.tg.failMethods["getChatMember"] = true

	hs.handle(commandUpdate(groupChatID, adminUser(), "/start"))

	wantTexts(t, hs.tg.sentTexts(), msgSomethingWentWrong)
}

func TestJoinCommand(t *testing.T) {
	hs := newHarness(t)

	hs.handle(commandUpdate(groupChatID, aliceUser(), "/join"))
	wantTexts(t, hs.tg.sentTexts(), msgNotActivated)

	hs.activate()
	hs.handle(commandUpdate(groupChatID, aliceUser(), "/join@"+botUsername))
	hs.handle(commandUpdate(groupChatID, aliceUser(), "/JOIN"))
	wantTexts(t, hs.tg.sentTexts(), msgEnrolled("Alice"), msgAlreadyEnrolled("Alice"))

	if s, err := hs.membership(aliceTGID); err != nil || s != domain.MembershipStatusActive {
		t.Fatalf("membership = %q, %v; want active", s, err)
	}
}

func TestJoinButton(t *testing.T) {
	hs := newHarness(t)
	hs.activate()

	hs.handle(callbackUpdate(*aliceUser(), joinCallbackData))
	answers := hs.tg.callsTo("answerCallbackQuery")
	if len(answers) != 1 || answers[0].fields["text"] != toastJoined || answers[0].fields["callback_query_id"] != "cq-1" {
		t.Fatalf("answers = %+v", answers)
	}
	wantTexts(t, hs.tg.sentTexts(), msgEnrolled("Alice"))

	hs.tg.reset()
	hs.handle(callbackUpdate(*aliceUser(), joinCallbackData))
	if a := hs.tg.callsTo("answerCallbackQuery"); len(a) != 1 || a[0].fields["text"] != toastAlreadyJoined {
		t.Fatalf("second answer = %+v", a)
	}
	if len(hs.tg.sentTexts()) != 0 {
		t.Fatalf("second tap posted %q; want no group message", hs.tg.sentTexts())
	}
}

func TestJoinButtonBeforeActivationAndUnknownData(t *testing.T) {
	hs := newHarness(t)

	hs.handle(callbackUpdate(*aliceUser(), joinCallbackData))
	if a := hs.tg.callsTo("answerCallbackQuery"); len(a) != 1 || a[0].fields["text"] != toastNotActivated {
		t.Fatalf("answer = %+v", a)
	}

	hs.tg.reset()
	hs.handle(callbackUpdate(*aliceUser(), "something-else"))
	if a := hs.tg.callsTo("answerCallbackQuery"); len(a) != 1 || a[0].fields["text"] != "" {
		t.Fatalf("unknown data must still be answered silently, got %+v", a)
	}
}

func TestCommandsForOtherBotsAndPlainTextAreIgnored(t *testing.T) {
	hs := newHarness(t)
	hs.activate()

	hs.handle(commandUpdate(groupChatID, aliceUser(), "/join@SomeOtherBot"))
	hs.handle(&models.Update{Message: &models.Message{Chat: groupChat(groupChatID), From: aliceUser(), Text: "join please"}})

	if calls := len(hs.tg.calls); calls != 0 {
		t.Fatalf("made %d API calls, want none", calls)
	}
}

func TestNewMembersAreInvitedNotEnrolled(t *testing.T) {
	hs := newHarness(t)
	hs.activate()

	hs.handle(&models.Update{Message: &models.Message{
		Chat:           groupChat(groupChatID),
		From:           aliceUser(),
		NewChatMembers: []models.User{*aliceUser(), {ID: 1003, FirstName: "Bo<b>"}, {ID: 555, IsBot: true, FirstName: "OtherBot"}},
	}})

	wantTexts(t, hs.tg.sentTexts(), msgMembersJoinedChat([]string{"Alice", "Bo&lt;b&gt;"}))
	if _, err := hs.membership(aliceTGID); !errors.Is(err, domain.ErrUserIdentityNotFound) {
		t.Fatalf("new member was enrolled automatically: %v", err)
	}
}

func TestNewMembersInInactiveGroupAreIgnored(t *testing.T) {
	hs := newHarness(t)
	hs.handle(&models.Update{Message: &models.Message{
		Chat: groupChat(groupChatID), From: aliceUser(), NewChatMembers: []models.User{*aliceUser()},
	}})
	if len(hs.tg.sentTexts()) != 0 {
		t.Fatalf("sent %q in an inactive group", hs.tg.sentTexts())
	}
}

func TestBotAddedWithoutDeepLinkGetsNudge(t *testing.T) {
	hs := newHarness(t)

	hs.handle(&models.Update{Message: &models.Message{
		Chat: groupChat(groupChatID), From: aliceUser(), NewChatMembers: []models.User{{ID: botID, IsBot: true, FirstName: "Tiny Cosmos"}},
	}})
	hs.h.Wait()

	wantTexts(t, hs.tg.sentTexts(), msgBotAdded)
}

func TestBotAddedThroughDeepLinkGetsNoNudge(t *testing.T) {
	hs := newHarness(t)

	hs.handle(&models.Update{Message: &models.Message{
		Chat: groupChat(groupChatID), From: adminUser(), NewChatMembers: []models.User{{ID: botID, IsBot: true, FirstName: "Tiny Cosmos"}},
	}})
	hs.handle(commandUpdate(groupChatID, adminUser(), "/start@"+botUsername+" "+StartGroupPayload))
	hs.h.Wait()

	for _, text := range hs.tg.sentTexts() {
		if text == msgBotAdded {
			t.Fatal("sent the /start hint even though the group was activated")
		}
	}
}

func TestNudgeCancelledOnShutdown(t *testing.T) {
	hs := newHarness(t)
	hs.h.nudgeDelay = time.Hour
	ctx, cancel := context.WithCancel(context.Background())

	hs.h.Handle(ctx, hs.api, &models.Update{Message: &models.Message{
		Chat: groupChat(groupChatID), From: aliceUser(), NewChatMembers: []models.User{{ID: botID, IsBot: true}},
	}})
	cancel()

	done := make(chan struct{})
	go func() { hs.h.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return after the context was cancelled")
	}
}

func TestMemberLeavingMarksMembershipLeft(t *testing.T) {
	hs := newHarness(t)
	hs.activate()
	hs.handle(commandUpdate(groupChatID, aliceUser(), "/join"))

	hs.handle(&models.Update{Message: &models.Message{Chat: groupChat(groupChatID), From: aliceUser(), LeftChatMember: aliceUser()}})

	if s, err := hs.membership(aliceTGID); err != nil || s != domain.MembershipStatusLeft {
		t.Fatalf("membership = %q, %v; want left", s, err)
	}
}

func TestGroupUpgradeToSupergroupKeepsGroup(t *testing.T) {
	hs := newHarness(t)
	hs.activate()
	const superID int64 = -1009999

	hs.handle(&models.Update{Message: &models.Message{Chat: groupChat(groupChatID), MigrateToChatID: superID}})
	super := models.Chat{ID: superID, Type: models.ChatTypeSupergroup}
	hs.handle(&models.Update{Message: &models.Message{Chat: super, MigrateFromChatID: groupChatID}})

	u := commandUpdate(superID, aliceUser(), "/join")
	u.Message.Chat = super
	hs.handle(u)
	wantTexts(t, hs.tg.sentTexts(), msgEnrolled("Alice"))
}

func TestPrivateStartExplainsAndLinksToGroups(t *testing.T) {
	hs := newHarness(t)

	hs.handle(commandUpdate(aliceTGID, aliceUser(), "/start"))

	wantTexts(t, hs.tg.sentTexts(), msgPrivateWelcome)
	kb := keyboardOf(t, hs.tg.callsTo("sendMessage")[0])
	if got := kb.InlineKeyboard[0][0].URL; got != "https://t.me/TinyCosmosBot?startgroup=activate" {
		t.Fatalf("add-to-group URL = %q", got)
	}
}

func TestHelp(t *testing.T) {
	hs := newHarness(t)

	hs.handle(commandUpdate(groupChatID, aliceUser(), "/help"))
	wantTexts(t, hs.tg.sentTexts(), msgHowItWorks+"\n\n"+msgNotActivated)

	hs.activate()
	hs.handle(commandUpdate(groupChatID, aliceUser(), "/help"))
	wantTexts(t, hs.tg.sentTexts(), msgHowItWorks)
}

func TestNamesAreHTMLEscaped(t *testing.T) {
	hs := newHarness(t)
	hs.activate()

	hs.handle(commandUpdate(groupChatID, &models.User{ID: 1004, FirstName: `<a href="x">Eve</a>`}, "/join"))

	wantTexts(t, hs.tg.sentTexts(), msgEnrolled("&lt;a href=&#34;x&#34;&gt;Eve&lt;/a&gt;"))
}

func TestAdminVerifier(t *testing.T) {
	tg := newFakeTelegram(t)
	v := NewAdminVerifier(tg.newBot())
	ctx := context.Background()

	for status, want := range map[string]bool{
		"creator": true, "administrator": true, "member": false, "restricted": false, "left": false, "kicked": false,
	} {
		tg.memberTypes["42"] = status
		got, err := v.IsTelegramGroupAdmin(ctx, groupChatID, 42)
		if err != nil || got != want {
			t.Errorf("status %q: got %v, %v; want %v", status, got, err, want)
		}
	}

	tg.failMethods["getChatMember"] = true
	if _, err := v.IsTelegramGroupAdmin(ctx, groupChatID, 42); err == nil {
		t.Fatal("want error when Telegram fails")
	}
}

func TestErrorsDoNotLeakToken(t *testing.T) {
	// Point the client at a closed server so the request itself fails.
	tg := newFakeTelegram(t)
	api := tg.newBot()
	tg.srv.Close()

	_, err := NewAdminVerifier(api).IsTelegramGroupAdmin(context.Background(), groupChatID, 42)
	if err == nil {
		t.Fatal("want error")
	}
	if strings.Contains(err.Error(), testToken) {
		t.Fatalf("error leaks the bot token: %v", err)
	}
}
