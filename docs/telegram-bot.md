# Telegram bot

The Telegram bot is the MVP interface for Tiny Cosmos. This page covers setting it up, running it locally, and what it does today.

## What it does

1. **Someone adds the bot to a group** with the "add to group" link: `https://t.me/<bot_username>?startgroup=activate`. Telegram lets them pick a group, adds the bot, and sends `/start` in that group for them.
2. **The bot checks with Telegram that the sender is a group admin.** If so, it creates the Tiny Cosmos group and enrolls that admin. If not, it asks for an admin to send `/start`.
3. **The bot posts a short onboarding sequence:** a welcome, how it works, and an invitation with an **✨ I'm in** button.
4. **Everyone else opts in** by tapping the button or sending `/join`. The bot creates their Tiny Cosmos user the first time.

Telegram doesn't let bots list a group's members, so nobody except the activating admin is enrolled automatically. That's also deliberate: taking part means sharing answers with the group, so it should be a choice.

| Situation | What the bot does |
|---|---|
| Bot added without the link | After a few seconds, if nobody has sent `/start`, it suggests an admin do so |
| `/start` again in an active group | Says it's already set up and shows the join button. Nobody is enrolled |
| `/start` from a non-admin | Explains that an admin needs to send it |
| `/start` from an anonymous admin | Asks them to send it from their personal account, since Telegram hides who they are |
| Someone joins the group later | Welcomes them and shows the join button. They aren't enrolled until they opt in |
| An enrolled member leaves the group | Marks their membership `left`. Rejoining with `/join` or the button makes it `active` again |
| The group is upgraded to a supergroup | Telegram changes the chat ID. The bot moves the group to the new ID, so nothing is lost |
| `/help` | Explains how it works (with the join button once the group is set up) |
| `/start` in a private chat | Explains Tiny Cosmos and offers the "add to group" button |

All message text lives in `internal/adapters/telegram/messages.go`.

## One-time setup with @BotFather

1. In Telegram, open **@BotFather** and send `/newbot`. Pick a display name (for example "Tiny Cosmos") and a username ending in `bot`.
2. Copy the token BotFather gives you.
3. Leave the defaults as they are: **Allow Groups** on, and **Group Privacy** on. With privacy on, the bot still receives everything it needs (commands, button taps, and join/leave notices) but never sees ordinary chat messages.

Treat the token like a password. Anyone who has it can control the bot. If it leaks, use `/revoke` in BotFather to get a new one.

## Running locally

From the repository root:

```bash
cp .env.example .env        # then paste your token into .env
go run ./cmd/bot
```

The bot reads `.env` from the directory you run it in, so run it from the repository root. Anything already set in your shell takes priority over `.env`. `.env` is git-ignored. On startup the bot creates `data/tiny-cosmos.db` if needed, applies migrations, registers its command menu with Telegram, and logs its "add to group" link. Stop it with Ctrl+C.

It uses long polling, so it needs outbound internet access but no public URL. Only run **one copy at a time** per token: Telegram rejects a second poller with a "conflict" error. The same error appears if a webhook was ever set for the bot; clear it by opening `https://api.telegram.org/bot<token>/deleteWebhook` once.

### Trying it out

1. Start the bot and open the `add_to_group_link` from its log on your phone.
2. Pick a group (or create a test group with a friend) and confirm.
3. You should see three messages, ending with the **✨ I'm in** button. You're already enrolled as the admin who activated it.
4. Have someone else tap the button or send `/join`.

To inspect the data, use any SQLite tool on `data/tiny-cosmos.db` (for example `sqlite3 data/tiny-cosmos.db 'SELECT * FROM group_memberships;'`).

## How it's built

| Layer | Location | Responsibility |
|---|---|---|
| Entry point | `cmd/bot` | Config, database, migrations, wiring, polling |
| Telegram adapter | `internal/adapters/telegram` | Turns updates into application calls and results into messages, using `github.com/go-telegram/bot` |
| Application | `internal/application` | Business rules: provisioning, activation (admin only), enrollment, leaving |
| Persistence | `internal/repositories/sqlite` | SQLite repositories and the migration runner |
| Domain | `internal/domain` | Users, identities, groups, integrations, memberships |

The application layer never imports the Telegram library. It asks "is this user a group admin?" through the `TelegramGroupAdminVerifier` interface, which the adapter implements with Telegram's `getChatMember`.

## Not done yet

- **Display names aren't stored.** The schema has no name column, so the bot uses Telegram first names only at the moment it sends a message. Prompt reveals will need names, either from Telegram at reveal time or from a new column.
- **Private chats for answering.** Telegram bots can only message people who have opened a private chat with them first. Before prompts can be sent privately, members need to be nudged to start a chat with the bot.
- **Leaving Tiny Cosmos without leaving the group** (a `/leave` command).
- **Removing the bot** keeps the group's data. Re-adding the bot resumes where it left off.
