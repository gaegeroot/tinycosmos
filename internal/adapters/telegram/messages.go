package telegram

import (
	"html"
	"strings"

	"github.com/go-telegram/bot/models"
)

// All user-facing copy lives here so it can be reviewed and edited in one
// place. Messages use Telegram's HTML parse mode; anything derived from user
// input must go through html.EscapeString (see displayName).

// joinCallbackData identifies the "I'm in" button.
const joinCallbackData = "join"

const (
	msgOnboardingWelcome = "🌌 <b>Tiny Cosmos is set up in this group!</b>\n\n" +
		"Once a week, everyone who joins gets the same question, something like " +
		"<i>“What made you laugh this week?”</i>"

	msgHowItWorks = "<b>How it works</b>\n" +
		"1. A new question arrives each week.\n" +
		"2. Answer on your own time. Nobody sees anyone's answer early.\n" +
		"3. At the weekly reveal, everyone's answers unlock together.\n\n" +
		"Missing a week is fine. Your cosmos only ever grows."

	msgAlreadyActive = "Tiny Cosmos is already set up here. " +
		"Haven't joined yet? Tap the button below or send /join."

	msgBotAdded = "👋 Thanks for adding me! A group admin can send /start to set up Tiny Cosmos here."

	msgNotAdmin = "Only a group admin can set up Tiny Cosmos here. Ask an admin to send /start."

	msgAnonymous = "I can't tell who sent that, because it was sent anonymously. " +
		"Please send it from your personal account."

	msgNotActivated = "Tiny Cosmos isn't set up in this group yet. A group admin can send /start to begin."

	msgSomethingWentWrong = "Something went wrong on my side. Please try again in a moment."

	msgPrivateWelcome = "🌌 <b>Hi, I'm Tiny Cosmos.</b>\n\n" +
		"I bring one question a week to your group chat. Everyone answers on their own time, " +
		"and the answers are revealed together.\n\n" +
		"Add me to a group to get started:"

	toastJoined        = "You're in! ✨"
	toastAlreadyJoined = "You're already in."
	toastNotActivated  = "Tiny Cosmos isn't set up here yet."
	toastError         = "Something went wrong. Please try again."
)

func msgJoinInvite(activatorName string) string {
	return "<b>Want in?</b> Tap the button below or send /join.\n\n" +
		activatorName + " is already in. Only people who join take part, so it's up to you."
}

func msgMembersJoinedChat(names []string) string {
	return "👋 Welcome, " + joinNames(names) + "! This group uses Tiny Cosmos: one question a week, " +
		"with everyone's answers revealed together. Tap the button below or send /join to take part."
}

func msgEnrolled(name string) string { return "✨ " + name + " joined the cosmos!" }

func msgAlreadyEnrolled(name string) string { return "You're already in, " + name + "." }

func joinKeyboard() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
		{Text: "✨ I'm in", CallbackData: joinCallbackData},
	}}}
}

func addToGroupKeyboard(link string) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
		{Text: "➕ Add Tiny Cosmos to a group", URL: link},
	}}}
}

// displayName returns an HTML-escaped name for a Telegram user.
func displayName(u *models.User) string {
	if u == nil {
		return "Someone"
	}
	name := strings.TrimSpace(u.FirstName)
	if name == "" && u.Username != "" {
		name = "@" + u.Username
	}
	if name == "" {
		name = "Someone"
	}
	return html.EscapeString(name)
}

func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	default:
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
}
