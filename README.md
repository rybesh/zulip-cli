# zulip-cli

[![CI](https://github.com/rybesh/zulip-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/rybesh/zulip-cli/actions/workflows/ci.yml)

A command-line interface and Go library for the Zulip API: scriptable access to
messages, channels, users, groups, emoji, linkifiers, profile fields, bot
storage, and the event stream. See [API Coverage](#api-coverage) for what is and
is not wrapped.

This began as a fork of [intelligrit/zulip-cli](https://github.com/intelligrit/zulip-cli)
and is now maintained separately here.

## Features

- ✅ **Every client method has a command** - the CLI exposes the whole library
- ✅ **Scriptable** - JSON on stdout, status and errors on stderr, safe to pipe to `jq`
- ✅ **Real-time events** - Listen for messages and events as they happen
- ✅ **Type-safe Go library** - Use in your own Go applications
- ✅ **No config files** - Simple environment variable authentication
- ✅ **Channel management** - Create, update, subscribe, manage topics
- ✅ **Message operations** - Send, fetch, update, delete, reactions
- ✅ **User management** - List, create, update, presence tracking
- ✅ **File uploads** - Upload and manage attachments

## Installation

```bash
go install github.com/rybesh/zulip-cli/cmd/zulip-cli@latest
```

Or build from source:

```bash
git clone https://github.com/rybesh/zulip-cli.git
cd zulip-cli
go build -o zulip-cli ./cmd/zulip-cli
```

## Quick Start

### Authentication

Set up your Zulip credentials as environment variables:

```bash
export ZULIP_URL=https://your-org.zulipchat.com
export ZULIP_EMAIL=bot@example.com
export ZULIP_API_KEY=your_api_key_here
```

You can get your API key from your Zulip account settings.

Optional settings:

```bash
export ZULIP_TIMEOUT=45s              # request timeout, default 15s
export ZULIP_CERT_BUNDLE=/path/ca.pem # trust a private certificate authority
export ZULIP_CLIENT_CERT=/path/client.pem
export ZULIP_CLIENT_CERT_KEY=/path/client.key
export ZULIP_INSECURE=true            # skip TLS verification (not recommended)
```

`--timeout` overrides `ZULIP_TIMEOUT` for a single invocation. File uploads on a
slow link are the usual reason to raise it.

For convenience, add to your shell profile:
```bash
# ~/.bashrc or ~/.zshrc
export ZULIP_URL=https://your-org.zulipchat.com
export ZULIP_EMAIL=bot@example.com
export ZULIP_API_KEY=your_api_key_here
```

### Basic Usage

```bash
# Send a message to a channel
zulip-cli send-message --channel general --topic "Hello" --content "Hi everyone!"

# Send a direct message
zulip-cli send-message --to user@example.com --content "Private message"

# List all channels
zulip-cli list-channels

# Get recent messages
zulip-cli get-messages --channel general --num-before 10

# Listen for new messages
zulip-cli listen
```

## Usage

Every command prints JSON on stdout; status lines and errors go to stderr, so
output stays safe to pipe. `--output` currently accepts only `json` —
YAML and table formats are on the roadmap, and asking for one is an error
rather than silently getting JSON.

Boolean flags are only sent when you pass them, so leaving one out means "use
the server's default" and passing `--flag=false` really does turn the setting
off. String flags work the same way: `--description ""` clears a description,
while omitting `--description` leaves it alone. Flag defaults shown in `--help`
match what the server does when the flag is absent.

Zulip accepts parameters it does not recognize and names them in the response.
When that happens, zulip-cli prints a warning on stderr and the JSON carries an
`ignored_parameters_unsupported` field, so a parameter your server version has
dropped fails loudly instead of silently doing nothing.

### Messages

```bash
# Send a channel message
zulip-cli send-message -s general -t "Announcements" -c "Important update!"

# Send a direct message to multiple users
zulip-cli send-message --to alice@example.com,bob@example.com -c "Meeting at 3pm"

# Get messages from a channel/topic
zulip-cli get-messages --channel engineering --topic "deployments" --num-before 20

# Filter by anything else the server understands
zulip-cli get-messages --narrow sender=alice@example.com --narrow is=unread

# Get the Markdown people typed, instead of the rendered HTML
zulip-cli get-messages --channel engineering --no-markdown
zulip-cli get-raw-message 12345

# Walk the whole history, a page at a time
zulip-cli get-messages --channel engineering --all --num-before 200

# Update a message
zulip-cli update-message 12345 --content "Updated text"

# Rename a whole thread, not just one message
zulip-cli update-message 12345 --topic "New Topic" --propagate-mode change_all

# Move one message to another channel
zulip-cli update-message 12345 --channel-id 43

# Delete a message
zulip-cli delete-message 12345

# Add emoji reaction. The name is always required; a code says which emoji
# a name belongs to, for a name more than one of them answers to.
zulip-cli add-reaction 12345 thumbs_up
zulip-cli add-reaction 12345 party-parrot --emoji-code 7 --reaction-type realm_emoji

# Mark specific messages read, unread, or starred
zulip-cli update-message-flags add read 12345 12346
zulip-cli update-message-flags remove read 12345
zulip-cli update-message-flags add starred 12345

# Upload a file, and list what you have uploaded
zulip-cli upload-file document.pdf
zulip-cli list-attachments

# Check what Markdown will look like without sending it
zulip-cli render-message "**hello** :tada:"

# Mark all messages as read
zulip-cli mark-all-as-read

# Get message edit history
zulip-cli get-message-history 12345
```

#### Narrowing

`get-messages` and `check-messages-match-narrow` take `--narrow
operator=operand`, repeatable, which reaches every filter the server
understands — not just the ones this CLI has heard of:

```bash
zulip-cli get-messages --narrow sender=alice@example.com
zulip-cli get-messages --narrow is=unread --narrow has=link
zulip-cli get-messages --narrow search="release notes"
zulip-cli get-messages --narrow dm=bob@example.com
```

`--channel` and `--topic` are shorthand for the two operators people reach for
most, and combine with `--narrow` rather than replacing it. Operands keep their
commas, so a search phrase stays one filter.

`check-messages-match-narrow` answers which of some messages match a narrow,
without fetching them:

```bash
zulip-cli check-messages-match-narrow 12345 12346 --narrow has=link
```

#### Message content and Markdown

Zulip renders message content to HTML by default. `--no-markdown` on
`get-messages` asks for the Markdown the sender typed instead, and
`get-raw-message` does the same for a single message. `--no-markdown=false`
asks for HTML explicitly, which is the server's default either way.

#### Paging through history

One `get-messages` request returns at most a page. `--all` keeps asking for
older messages, `--num-before` at a time, until the server says the history has
run out, and prints one merged result — so walking a channel's history no
longer means re-anchoring by hand on the oldest ID and watching `found_oldest`.
It walks backwards only, so it cannot be combined with `--num-after`.

### Channels

Zulip renamed **streams** to **channels** in version 9.0. This CLI uses the
current name; every old `*-stream*` command and the `--stream` flag still work as
aliases, so existing scripts keep running. Response JSON belongs to the server and
is unchanged — keys such as `stream_id` and `streams` stay exactly as they are.

```bash
# List all channels
zulip-cli list-channels

# Get a channel, by name or by ID
zulip-cli get-channel "general"

# Get only a channel's ID, by name
zulip-cli get-channel-id "general"

# Create a channel
zulip-cli create-channel engineering --description "Engineering discussions"

# Create a channel with other people in it, and settings applied up front
zulip-cli create-channel announcements \
    --subscribers alice@example.com,17 \
    --can-send-message-group role:administrators \
    --invite-only

# Update channel settings
zulip-cli update-channel 42 --description "New description"

# Make a channel private, or public again
zulip-cli update-channel 42 --invite-only --history-public-to-subscribers
zulip-cli update-channel 42 --invite-only=false

# Change how long a channel keeps messages
zulip-cli update-channel 42 --message-retention-days 30
zulip-cli update-channel 42 --message-retention-days unlimited

# Change who may post in a channel
zulip-cli update-channel 42 --can-send-message-group role:moderators

# Delete a channel
zulip-cli delete-channel 42

# Subscribe to channels
zulip-cli subscribe engineering design product

# Subscribe other people, by ID or by email
zulip-cli subscribe engineering --principals 17,alice@example.com

# Unsubscribe from channels, yourself or others
zulip-cli unsubscribe random
zulip-cli unsubscribe random --principals alice@example.com

# Check whether someone is subscribed
zulip-cli get-subscription-status alice@example.com engineering

# List your subscriptions
zulip-cli list-subscriptions

# List only the channels you are subscribed to
zulip-cli list-channels --include-public=false

# Change your own settings for a channel
zulip-cli update-subscription engineering --color "#76ce90" --pin-to-top
zulip-cli update-subscription engineering --is-muted=false

# Get the address that emails messages into a channel
zulip-cli get-channel-email-address engineering

# Subscribe new users to a channel automatically
zulip-cli add-default-channel engineering

# Mute a topic, by channel name or by channel ID
zulip-cli mute-topic general "off-topic"
zulip-cli unmute-topic 42 "off-topic"

# Follow a topic
zulip-cli follow-topic general "release planning"
zulip-cli unfollow-topic general "release planning"

# Set any visibility policy: inherit, muted, unmuted, or followed
zulip-cli set-topic-visibility general "off-topic" unmuted

# Move topic to another channel
zulip-cli move-topic 42 "old-name" --new-channel-id 43
```

#### Channel settings and your own settings

`update-channel` changes the channel, for everyone. `update-subscription`
changes your own preferences for it — its colour, whether it is pinned or
muted, and which notifications it sends — and changes nothing for anyone else.

`--invite-only`, `--is-web-public` and `--history-public-to-subscribers` are
tri-state on `update-channel`: leaving one out changes nothing, and
`--invite-only=false` makes a private channel public rather than being read as
"leave it alone". Changing a channel's privacy may also need
`--history-public-to-subscribers`, since the server decides what happens to the
existing history from the two together.

`subscribe` and `unsubscribe` act on you unless `--principals` names other
people, by user ID or email address. Subscribing someone you may not add fails
the whole request; `--authorization-errors-fatal=false` subscribes everyone
allowed instead and reports the rest under `unauthorized`.

#### Topic visibility

Your personal preference for a topic is one of four policies:

- `inherit` — no policy of its own; the topic follows its channel. Also
  spelled `none`.
- `muted` — hide the topic.
- `unmuted` — show the topic even though its channel is muted.
- `followed` — follow the topic.

`mute-topic`, `unmute-topic`, `follow-topic` and `unfollow-topic` are shorthand
for `set-topic-visibility` with the matching policy; unmuting and unfollowing
both clear the policy, which is what the server does either way.

These commands use `POST /user_topics`, added in feature level 170. Servers
older than that fall back to the endpoint it deprecates, which can only mute and
unmute — `unmuted` and `followed` are refused there rather than being turned
into the nearest thing it understands. `followed` itself needs feature level
219.

#### Channel permissions

Every `--can-*-group` flag takes a user group: an ID from `list-user-groups`, or
a name — including the `role:` system groups every organization has, such as
`role:everyone`, `role:members`, `role:moderators`, `role:administrators`, and
`role:nobody`. Run `create-channel --help` for the full list of permissions.

`--can-send-message-group` replaces the `stream_post_policy` setting Zulip
removed in feature level 333. Channels still report a `stream_post_policy` and an
`is_announcement_only` in responses, but since that removal the server computes
them from `can_send_message_group` as the nearest enclosing role, so they are an
approximation of who may post rather than the setting itself.

`create-channel` uses `POST /channels/create` on servers from feature level 417,
and creates the channel by subscribing to it on older ones. Both accept the same
settings, so the choice only shows in the response: the newer endpoint reports
the new channel's `id`, the older one reports who was subscribed.

### Users

```bash
# List all users
zulip-cli list-users

# Get user details, by ID or by email address
zulip-cli get-user 123
zulip-cli get-user dana@example.com

# Get your own profile
zulip-cli get-profile

# Create a user
zulip-cli create-user user@example.com "Full Name"

# Update user
zulip-cli update-user 123 --full-name "New Name"

# Set custom profile fields, by field ID
zulip-cli update-user 123 --profile-data 4=Berlin --profile-data 6="Team lead"

# Get user presence, for one user or everyone
zulip-cli get-user-presence 123
zulip-cli list-presence

# Update your presence
zulip-cli update-presence active

# Tell people you are typing
zulip-cli set-typing-status start --to alice@example.com
zulip-cli set-typing-status stop --channel general --topic standup

# Change your account-wide notification settings
zulip-cli update-notification-settings --channel-desktop=false --offline-push
```

The field IDs `--profile-data` takes are the ones `list-profile-fields`
reports. Notification flags are tri-state, like the channel privacy settings:
leaving one out changes nothing, and `--flag=false` turns that notification
off. `update-notification-settings` is your whole account;
`update-subscription` is one channel.

### Other Commands

```bash
# Print version information
zulip-cli version          # structured, honors --output
zulip-cli --version        # single line

# Get server settings
zulip-cli server-settings

# List custom emoji
zulip-cli list-emoji

# Upload custom emoji
zulip-cli upload-emoji smiley emoji.png

# Manage alert words
zulip-cli list-alert-words
zulip-cli add-alert-words "urgent" "asap" "critical"

# Manage user groups. Name and description are both positional; --members
# takes user IDs, which list-users reports.
zulip-cli list-user-groups
zulip-cli create-user-group "Engineering" "Engineering team" --members 12,34
zulip-cli update-user-group 7 --name "Platform" --description "Platform team"
zulip-cli add-group-members 7 56 78
zulip-cli remove-group-members 7 56
zulip-cli delete-user-group 7

# Manage linkifiers, which turn patterns in messages into links
zulip-cli list-linkifiers
zulip-cli add-linkifier '#(?P<id>[0-9]+)' 'https://example.com/issues/{id}'
zulip-cli remove-linkifier 7

# Manage custom profile fields
zulip-cli list-profile-fields
zulip-cli create-profile-field Location --field-type 1 --hint "Where you work"
zulip-cli update-profile-field 4 --hint "City you work from"
zulip-cli reorder-profile-fields 6 4 5
zulip-cli delete-profile-field 4

# Read and write this bot's stored state
zulip-cli get-storage
zulip-cli update-storage last-seen=1717171717 greeting="hello there"

# Watch for new messages, printed for a human to read
zulip-cli listen

# Watch the raw event stream as JSON, one event per value
zulip-cli listen --event-types message
zulip-cli listen --event-types message,reaction,subscription

# Drive a queue by hand, instead of holding one open with listen
zulip-cli register --event-types message
zulip-cli get-events 1518familiar --last-event-id -1
zulip-cli get-events 1518familiar --last-event-id 7 --dont-block

# Release an event queue a killed listener left behind
zulip-cli deregister 1518familiar
```

`listen` has two modes, and both print JSON, one value after another. With no
`--event-types` (or with `--messages-only`) each value is a **message**, so the
fields are `.sender_full_name`, `.content`, and so on. With `--event-types`
each value is a **raw event**, which nests a message under `.message`: there
the fields are `.message.sender_full_name`, `.message.content`. The two flags
cannot be combined.

`listen` keeps an event queue of its own. `register`, `get-events` and
`deregister` are the same queue taken apart, for a script that wants to poll on
its own schedule: register once, then pass each fetch the largest event id the
last one returned, so no event is read twice.

Bot storage belongs to the bot whose credentials are in the environment, so a
human account has none. `listen` releases its own queue when stopped with
Ctrl-C; `deregister` is for the queues left by a listener that was killed
outright.

## JSON Output & jq Examples

Every command prints JSON on stdout, so the examples below pipe straight into
`jq`.

### Pretty Print

```bash
# Pretty print all channels
zulip-cli list-channels | jq

# Pretty print with color
zulip-cli list-users | jq -C
```

### Extracting Data

```bash
# Get just channel names
zulip-cli list-channels | jq -r '.streams[].name'

# Get email addresses of all users
zulip-cli list-users | jq -r '.members[].email'

# Extract message content
zulip-cli get-messages --channel general --num-before 10 | jq -r '.messages[].content'

# Count total messages
zulip-cli get-messages --channel general --num-before 100 | jq '.messages | length'
```

### Filtering with jq

```bash
# Filter channels by name pattern
zulip-cli list-channels | jq '.streams[] | select(.name | contains("eng"))'

# Get messages from specific sender
zulip-cli get-messages --channel general --num-before 50 | \
  jq '.messages[] | select(.sender_full_name == "Alice")'

# Find messages with reactions
zulip-cli get-messages --channel general --num-before 100 | \
  jq '.messages[] | select(.reactions | length > 0)'

# List admin users
zulip-cli list-users | jq '.members[] | select(.is_admin == true) | .full_name'

# Count users by type
zulip-cli list-users | jq 'group_by(.is_bot) | map({bot: .[0].is_bot, count: length})'

# Get topics with recent activity
zulip-cli list-channel-topics 42 | jq '.topics | sort_by(.max_id) | reverse | .[0:5]'
```

### Format Conversions

```bash
# Format messages as CSV
zulip-cli get-messages --channel general --num-before 10 | \
  jq -r '.messages[] | [.id, .sender_full_name, .subject, .content] | @csv'

# Create a table of channels
zulip-cli list-channels | \
  jq -r '.streams[] | "\(.stream_id)\t\(.name)\t\(.description)"' | column -t

# Export to YAML
zulip-cli get-profile | yq -P
```

### Scripting Examples

**Send daily digest:**
```bash
#!/bin/bash
CHANNEL="general"
TOPIC="Daily Digest"
DATE=$(date +%Y-%m-%d)

# Get today's messages
MESSAGES=$(zulip-cli get-messages --channel "$CHANNEL" --num-before 100 | \
  jq -r ".messages[] | select(.timestamp > $(date -d 'today' +%s)) | \
    \"- [\(.sender_full_name)]: \(.content)\""
)

# Send digest
zulip-cli send-message \
  --channel "$CHANNEL" \
  --topic "$TOPIC" \
  --content "**Digest for $DATE**\n\n$MESSAGES"
```

**Monitor for mentions:**
```bash
#!/bin/bash
MY_NAME="Alice"

# Event mode is the one that emits JSON; message mode prints text for a human.
zulip-cli listen --event-types message | jq --unbuffered -r \
  ".message | select(.sender_full_name != \"$MY_NAME\" and (.content | contains(\"@$MY_NAME\"))) | \
   \"[MENTION] \(.sender_full_name) in #\(.display_recipient)/\(.subject): \(.content)\""
```

**Bulk subscribe users:**
```bash
#!/bin/bash
CHANNEL="announcements"

# Get all active user emails
USER_EMAILS=$(zulip-cli list-users | \
  jq -r '.members[] | select(.is_active == true and .is_bot == false) | .email')

# Subscribe them all
for email in $USER_EMAILS; do
  echo "Subscribing $email to $CHANNEL"
  zulip-cli subscribe "$CHANNEL" --principals "$email"
done
```

**Generate message statistics:**
```bash
#!/bin/bash
CHANNEL="general"
MESSAGES=$(zulip-cli get-messages --channel "$CHANNEL" --num-before 500)

echo "Message Statistics for #$CHANNEL"
echo "================================"
echo -n "Total messages: "
echo "$MESSAGES" | jq '.messages | length'

echo -n "Unique senders: "
echo "$MESSAGES" | jq '[.messages[].sender_full_name] | unique | length'

echo "Top 5 senders:"
echo "$MESSAGES" | jq -r '[.messages | group_by(.sender_full_name) | .[] |
  {sender: .[0].sender_full_name, count: length}] |
  sort_by(.count) | reverse | .[0:5] | .[] |
  "  \(.sender): \(.count) messages"'
```

**Watch for keywords:**
```bash
#!/bin/bash
KEYWORDS=("urgent" "critical" "help")

zulip-cli listen --event-types message | jq --unbuffered -r \
  ".message | select(.content | ascii_downcase | test(\"$(IFS='|'; echo "${KEYWORDS[*]}")\")) | \
   \"[ALERT] \(.sender_full_name): \(.content)\"" | \
  while read -r alert; do
    echo "$alert"
    osascript -e "display notification \"$alert\" with title \"Zulip Alert\""
  done
```

## Using as a Go Library

### Installation

```bash
go get github.com/rybesh/zulip-cli
```

### Example Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/rybesh/zulip-cli/client"
    "github.com/rybesh/zulip-cli/types"
)

func main() {
    // Create client (uses environment variables)
    c, err := client.NewClient()
    if err != nil {
        log.Fatal(err)
    }

    // Send a message
    resp, err := c.SendMessage(client.SendMessageRequest{
        Type:    "stream",
        To:      "general",
        Topic:   "Hello",
        Content: "Hi from Go!",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Message sent! ID: %d\n", resp.ID)

    // Get messages
    messages, err := c.GetMessages(client.GetMessagesRequest{
        Anchor:    "newest",
        NumBefore: 10,
        NumAfter:  0,
        Narrow: []types.Narrow{
            {Operator: "stream", Operand: "general"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, msg := range messages.Messages {
        fmt.Printf("[%s] %s: %s\n", msg.Subject, msg.SenderFullName, msg.Content)
    }
}
```

### Custom Configuration

```go
config := client.Config{
    URL:      "https://your-org.zulipchat.com",
    Email:    "bot@example.com",
    APIKey:   "your_api_key",
    Verbose:  true,            // Enable debug output
    Insecure: false,           // Set true to skip TLS verification (not recommended)
    Timeout:  45 * time.Second, // Zero means the 15s default

    // For servers behind a private CA or requiring a client certificate
    CertBundle:    "/path/ca.pem",
    ClientCert:    "/path/client.pem",
    ClientCertKey: "/path/client.key",
}

c, err := client.NewClientWithConfig(config)
```

Constructing a client does not contact the server. Credential problems surface
as an `*client.APIError` from the first real request:

```go
if _, err := c.GetProfile(); err != nil {
    var apiErr *client.APIError
    if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized {
        log.Fatal("bad credentials")
    }
}
```

### Event Streaming

Both listeners run until the context is cancelled, retry transient failures
with backoff, and deregister their event queue on the way out.

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

// Listen for all events
err := c.CallOnEachEvent(ctx, func(event map[string]interface{}) {
    fmt.Printf("Event: %v\n", event)
}, []string{"message", "reaction"}, nil)

// Listen for messages only
err := c.CallOnEachMessage(ctx, func(msg types.Message) {
    fmt.Printf("Message from %s: %s\n", msg.SenderFullName, msg.Content)
})
```

## Project Structure

```
zulip-cli/
├── client/              # Go client library
│   ├── client.go        # Core client, auth, request plumbing
│   ├── messages.go      # Message operations
│   ├── streams.go       # Channel, subscription, and topic operations
│   ├── users.go         # User operations
│   ├── groups.go        # User group operations
│   ├── realm.go         # Emoji, linkifiers, profile fields
│   ├── events.go        # Event queues and streaming
│   ├── storage.go       # Bot storage
│   └── version.go       # Client version string
├── types/               # Type definitions
│   └── types.go         # All API types
├── cmd/zulip-cli/       # CLI application
│   ├── main.go
│   └── commands/        # One file per command group, mirroring client/
│       ├── root.go      # Root command, flags, output, flag helpers
│       ├── messages.go
│       ├── streams.go
│       ├── users.go
│       ├── groups.go
│       ├── realm.go
│       ├── storage.go
│       ├── misc.go      # Alert words, listen, deregister, server settings
│       └── version.go
├── .github/workflows/   # CI
├── LICENSE
├── README.md
└── go.mod
```

## Development

### Prerequisites

- Go 1.23 or higher
- Access to a Zulip server for testing

### Building

```bash
go build -o zulip-cli ./cmd/zulip-cli
```

### Testing

The test suite needs no Zulip server: it runs the client and the commands
against an `httptest` server and asserts on the requests they produce.

```bash
go test ./...
go test -race ./...
```

To try the binary against a real server:

```bash
export ZULIP_URL=https://your-test-org.zulipchat.com
export ZULIP_EMAIL=test-bot@example.com
export ZULIP_API_KEY=your_test_api_key

./zulip-cli server-settings
./zulip-cli get-profile
./zulip-cli list-channels
```

### Continuous integration

`.github/workflows/ci.yml` runs `gofmt -l .`, `go build ./...`, `go vet ./...`,
and `go test -race ./...` on every push to `main` and on every pull request.

## API Coverage

This is not a complete wrapper of the Zulip API, and the parts it does wrap are
the parts listed here. Every method the client library exposes has a command
attached to it, so anything below is reachable from both Go and the shell.

Wrapped:

- **Messages** - send, fetch, raw Markdown, update, delete, edit history,
  reactions, flags, mark-as-read, render, narrow matching
- **Channels** - get, list, create, update, delete, topics, email address,
  subscribe, unsubscribe, subscribers, subscription settings, default channels
- **Topics** - move, rename, and set a visibility policy (mute, unmute, follow)
- **Users** - list, get, profile, create, update, deactivate, reactivate,
  presence, alert words, typing status, notification settings
- **User Groups** - list, create, update, delete, add and remove members
- **Emoji** - list, upload, and delete custom emoji
- **Realm** - linkifiers, custom profile fields, server settings
- **Events** - register a queue, poll it, stream events or messages, deregister
- **Files** - upload attachments and list your own
- **Bot storage** - read and write the calling bot's stored state

Not wrapped, and the most likely candidates if you need more: drafts, scheduled
messages, read receipts, user status, saved snippets, invitations, outgoing
webhooks and bot management, realm export and settings administration.

## Roadmap

Tracked as issues in this repository. The larger open items:

- [ ] YAML output format (`-o yaml`)
- [ ] Table output format (`-o table`)
- [ ] Optional configuration file, as an alternative to environment variables
- [ ] The unwrapped endpoint groups listed under API Coverage

Shell completion is already available through Cobra: `zulip-cli completion bash`
(or `zsh`, `fish`, `powershell`).

## License

MIT. See [LICENSE](LICENSE).

The project began as a fork of
[intelligrit/zulip-cli](https://github.com/intelligrit/zulip-cli), and the
original copyright notice stays in `LICENSE` alongside ours.

## Contributing

Issues and pull requests are welcome at
[github.com/rybesh/zulip-cli](https://github.com/rybesh/zulip-cli).

Before opening a pull request:

1. `gofmt -l .` reports nothing
2. `go vet ./...` is clean
3. `go test -race ./...` passes
4. New behavior comes with a test — the suite runs against `httptest`, so it
   needs no Zulip server

CI runs all four on every push and pull request. Beyond that: exported
functions and types get doc comments, and commit messages say what changed and
why.

## Support

Open an issue at
[github.com/rybesh/zulip-cli/issues](https://github.com/rybesh/zulip-cli/issues).

## Acknowledgments

- Built with the [Cobra](https://github.com/spf13/cobra) CLI framework
- Modeled on [python-zulip-api](https://github.com/zulip/python-zulip-api)
- Thanks to the Zulip team for their excellent API documentation
