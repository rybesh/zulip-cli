# zulip-cli

**An [Intelligrit Labs](https://intelligrit.com#labs) Project**

<p align="center">
  <img src="logo.png" alt="zulip-cli logo" width="200">
</p>

A comprehensive command-line interface and Go library for the Zulip API. Provides complete scriptable access to messages, channels, users, and all Zulip features with full API coverage.

## Features

- ✅ **Complete API coverage** - Every Zulip endpoint supported
- ✅ **Scriptable** - JSON output by default, perfect for piping to `jq` or other tools
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
zulip-cli listen --messages-only
```

## Usage

### Messages

```bash
# Send a channel message
zulip-cli send-message -s general -t "Announcements" -c "Important update!"

# Send a direct message to multiple users
zulip-cli send-message --to alice@example.com,bob@example.com -c "Meeting at 3pm"

# Get messages from a channel/topic
zulip-cli get-messages --channel engineering --topic "deployments" --num-before 20

# Update a message
zulip-cli update-message 12345 --content "Updated text"

# Change message topic
zulip-cli update-message 12345 --topic "New Topic"

# Delete a message
zulip-cli delete-message 12345

# Add emoji reaction
zulip-cli add-reaction 12345 thumbs_up

# Upload a file
zulip-cli upload-file document.pdf

# Mark all messages as read
zulip-cli mark-all-as-read

# Get message edit history
zulip-cli get-message-history 12345
```

### Channels

Zulip renamed **streams** to **channels** in version 9.0. This CLI uses the
current name; every old `*-stream*` command and the `--stream` flag still work as
aliases, so existing scripts keep running. Response JSON belongs to the server and
is unchanged — keys such as `stream_id` and `streams` stay exactly as they are.

```bash
# List all channels
zulip-cli list-channels

# Get channel ID by name
zulip-cli get-channel "general"

# Create a channel
zulip-cli create-channel engineering --description "Engineering discussions"

# Update channel settings
zulip-cli update-channel 42 --description "New description"

# Delete a channel
zulip-cli delete-channel 42

# Subscribe to channels
zulip-cli subscribe engineering design product

# Unsubscribe from channels
zulip-cli unsubscribe random

# List your subscriptions
zulip-cli list-subscriptions

# Mute a topic
zulip-cli mute-topic --channel general --topic "off-topic"

# Move topic to another channel
zulip-cli move-topic --channel-id 42 --new-channel-id 43 --topic "old-name"
```

### Users

```bash
# List all users
zulip-cli list-users

# Get user details
zulip-cli get-user 123

# Get your own profile
zulip-cli get-profile

# Create a user
zulip-cli create-user user@example.com "Full Name"

# Update user
zulip-cli update-user 123 --full-name "New Name"

# Get user presence
zulip-cli get-user-presence 123

# Update your presence
zulip-cli update-presence active
```

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

# Manage user groups
zulip-cli list-user-groups
zulip-cli create-user-group "Engineering" --description "Engineering team"
```

## JSON Output & jq Examples

All commands output JSON by default, making zulip-cli perfect for scripting.

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

zulip-cli listen --messages-only | jq --unbuffered -r \
  "select(.sender_full_name != \"$MY_NAME\" and (.content | contains(\"@$MY_NAME\"))) | \
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

zulip-cli listen --messages-only | jq --unbuffered -r \
  "select(.content | ascii_downcase | test(\"$(IFS='|'; echo "${KEYWORDS[*]}")\")) | \
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
    Verbose:  true,  // Enable debug output
    Insecure: false, // Set true to skip TLS verification (not recommended)
}

c, err := client.NewClientWithConfig(config)
```

### Event Streaming

```go
// Listen for all events
err := c.CallOnEachEvent(func(event map[string]interface{}) {
    fmt.Printf("Event: %v\n", event)
}, []string{"message", "reaction"}, nil)

// Listen for messages only
err := c.CallOnEachMessage(func(msg types.Message) {
    fmt.Printf("Message from %s: %s\n", msg.SenderFullName, msg.Content)
})
```

## Project Structure

```
zulip-cli/
├── client/              # Go client library
│   ├── client.go        # Core client & auth
│   ├── messages.go      # Message operations
│   ├── streams.go       # Stream operations
│   ├── users.go         # User operations
│   ├── groups.go        # User group operations
│   ├── realm.go         # Realm/emoji/filters
│   ├── events.go        # Event streaming
│   └── storage.go       # Bot storage
├── types/               # Type definitions
│   └── types.go         # All API types
├── cmd/zulip-cli/       # CLI application
│   ├── main.go
│   └── commands/        # CLI commands
│       ├── root.go
│       ├── messages.go
│       ├── streams.go
│       ├── users.go
│       ├── groups.go
│       └── misc.go
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

```bash
# Set up test environment
export ZULIP_URL=https://your-test-org.zulipchat.com
export ZULIP_EMAIL=test-bot@example.com
export ZULIP_API_KEY=your_test_api_key

# Test basic commands
./zulip-cli server-settings
./zulip-cli get-profile
./zulip-cli list-channels

# Run Go tests
go test ./...
```

## API Coverage

The library provides complete coverage of the Zulip API including:

- **Messages** - Send, fetch, update, delete, reactions, flags, rendering
- **Channels** - Create, update, delete, subscribe, topics, email addresses
- **Users** - List, create, update, deactivate, presence, alert words
- **User Groups** - Create, update, delete, manage members
- **Emoji** - List, upload, delete custom emoji
- **Realm** - Linkifiers, profile fields, server settings
- **Events** - Real-time event streaming and message listening
- **Files** - Upload and manage attachments

## Roadmap

- [ ] YAML output format support
- [ ] Table output format for better CLI readability
- [ ] Configuration file support (optional)
- [ ] Shell completion scripts
- [ ] Webhooks/outgoing webhooks support
- [ ] Message drafts management
- [ ] Typing indicators
- [ ] Read receipts
- [ ] OPML export for subscriptions

## About Intelligrit Labs

zulip-cli is developed by [Intelligrit Labs](https://intelligrit.com#labs), the R&D arm of Intelligrit LLC. We build tools for ourselves and release them for everyone. Intelligrit delivers AI-driven IT modernization for federal agencies.

## License

MIT License - see [LICENSE](LICENSE) file for details

## Contributing

Contributions are welcome! This project follows standard Go conventions.

### Guidelines

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following Go best practices
4. Write tests for new functionality
5. Ensure all tests pass (`go test ./...`)
6. Run `go fmt` and `go vet`
7. Commit your changes (`git commit -m 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

### Code Style

- Follow standard Go formatting (`gofmt`, `go vet`)
- Write clear, descriptive commit messages
- Add comments for exported functions and types
- Keep functions focused and modular

## Support

For issues, questions, or contributions, please open an issue on GitHub.

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Complete reimplementation of the [python-zulip-api](https://github.com/zulip/python-zulip-api) library in Go
- Thanks to the Zulip team for their excellent API documentation
