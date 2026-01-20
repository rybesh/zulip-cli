# zulip-cli

**An [Intelligrit Labs](https://intelligrit.com#labs) Project**

A comprehensive command-line interface and Go library for the Zulip API. Provides complete scriptable access to messages, streams, users, and all Zulip features with full API coverage.

## Features

- Complete Zulip API coverage - every endpoint supported
- Send and manage messages with rich formatting
- Create and manage streams and subscriptions
- User and user group management
- Real-time event streaming and message listening
- File uploads and attachment management
- Custom emoji and realm configuration
- Fully scriptable with JSON output - perfect for automation
- Type-safe Go library for building Zulip integrations
- No config files needed - uses environment variables only

## Installation

### From Source

```bash
go install github.com/intelligrit/zulip-cli/cmd/zulip-cli@latest
```

### Build Locally

```bash
git clone https://github.com/intelligrit/zulip-cli.git
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

### Basic Usage

Send a message to a stream:

```bash
zulip-cli send-message --stream general --topic "Hello" --content "Hi everyone!"
```

Send a direct message:

```bash
zulip-cli send-message --to user1@example.com,user2@example.com --content "Private message"
```

List all streams:

```bash
zulip-cli list-streams
```

Get recent messages from a stream:

```bash
zulip-cli get-messages --stream general --num-before 10
```

## Commands

### Messages

```bash
# Send a stream message
zulip-cli send-message -s general -t "Announcements" -c "Important update!"

# Send a direct message to multiple users
zulip-cli send-message --to alice@example.com,bob@example.com -c "Meeting at 3pm"

# Get messages from a stream/topic
zulip-cli get-messages --stream engineering --topic "deployments" --num-before 20

# Update a message
zulip-cli update-message 12345 --content "Updated text"

# Change message topic
zulip-cli update-message 12345 --topic "New Topic"

# Delete a message
zulip-cli delete-message 12345

# Add emoji reaction
zulip-cli add-reaction 12345 thumbs_up

# Remove emoji reaction
zulip-cli remove-reaction 12345 thumbs_up

# Upload a file
zulip-cli upload-file document.pdf

# Mark all messages as read
zulip-cli mark-all-as-read

# Mark stream as read
zulip-cli mark-stream-as-read 42

# Mark topic as read
zulip-cli mark-topic-as-read 42 "topic-name"

# Get message edit history
zulip-cli get-message-history 12345
```

### Streams

```bash
# List all streams
zulip-cli list-streams

# Get stream ID by name
zulip-cli get-stream "general"

# Create a stream
zulip-cli create-stream engineering --description "Engineering discussions"

# Update stream settings
zulip-cli update-stream 42 --description "New description"

# Delete a stream
zulip-cli delete-stream 42

# List topics in a stream
zulip-cli list-stream-topics 42

# Subscribe to streams
zulip-cli subscribe engineering design product

# Unsubscribe from streams
zulip-cli unsubscribe random

# List your subscriptions
zulip-cli list-subscriptions

# List stream subscribers
zulip-cli list-subscribers 42

# Mute a topic
zulip-cli mute-topic --stream general --topic "off-topic"

# Unmute a topic
zulip-cli unmute-topic --stream general --topic "off-topic"

# Move topic to another stream
zulip-cli move-topic --stream-id 42 --new-stream-id 43 --topic "old-name" --new-topic "new-name"
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

# Deactivate user
zulip-cli deactivate-user 123

# Reactivate user
zulip-cli reactivate-user 123

# Get user presence
zulip-cli get-user-presence 123

# Update your presence
zulip-cli update-presence active
```

### User Groups

```bash
# List all user groups
zulip-cli list-user-groups

# Create a user group
zulip-cli create-user-group "Engineering" --description "Engineering team" --members 1,2,3

# Update user group
zulip-cli update-user-group 5 --name "Backend Team"

# Delete user group
zulip-cli delete-user-group 5

# Add members to group
zulip-cli add-group-members 5 --add 10,11,12

# Remove members from group
zulip-cli remove-group-members 5 --remove 10
```

### Other

```bash
# Get server settings
zulip-cli server-settings

# List custom emoji
zulip-cli list-emoji

# Upload custom emoji
zulip-cli upload-emoji smiley emoji.png

# Delete custom emoji
zulip-cli delete-emoji smiley

# List alert words
zulip-cli list-alert-words

# Add alert words
zulip-cli add-alert-words "urgent" "asap" "critical"

# Remove alert words
zulip-cli remove-alert-words "urgent"

# Listen for new messages (runs continuously)
zulip-cli listen --messages-only
```

## JSON Output and jq

All commands output JSON format for easy parsing and scripting. The output is formatted with 2-space indentation for human readability while remaining machine-parseable.

### Pretty Printing

For even prettier output, pipe through `jq`:

```bash
zulip-cli list-streams | jq
```

### jq Examples

#### Get just stream names

```bash
zulip-cli list-streams | jq -r '.streams[].name'
```

#### Filter streams by name pattern

```bash
zulip-cli list-streams | jq '.streams[] | select(.name | contains("eng"))'
```

#### Count total messages

```bash
zulip-cli get-messages --stream general --num-before 100 | jq '.messages | length'
```

#### Get messages from specific sender

```bash
zulip-cli get-messages --stream general --num-before 50 | jq '.messages[] | select(.sender_full_name == "Alice")'
```

#### Extract just message content

```bash
zulip-cli get-messages --stream general --num-before 10 | jq -r '.messages[].content'
```

#### Get unread message count by stream

```bash
zulip-cli list-subscriptions | jq '.subscriptions[] | {stream: .name, unread: (.is_muted | not)}'
```

#### Find messages with specific reactions

```bash
zulip-cli get-messages --stream general --num-before 100 | jq '.messages[] | select(.reactions | length > 0) | {id, content, reactions}'
```

#### List users with specific role

```bash
zulip-cli list-users | jq '.members[] | select(.is_admin == true) | .full_name'
```

#### Get email addresses of all users

```bash
zulip-cli list-users | jq -r '.members[].email'
```

#### Count users by bot vs human

```bash
zulip-cli list-users | jq 'group_by(.is_bot) | map({bot: .[0].is_bot, count: length})'
```

#### Find streams you're subscribed to

```bash
zulip-cli list-subscriptions | jq -r '.subscriptions[].name'
```

#### Get topics with recent activity

```bash
zulip-cli list-stream-topics 42 | jq '.topics | sort_by(.max_id) | reverse | .[0:5]'
```

#### Format messages as CSV

```bash
zulip-cli get-messages --stream general --num-before 10 | jq -r '.messages[] | [.id, .sender_full_name, .subject, .content] | @csv'
```

#### Find messages with attachments

```bash
zulip-cli get-messages --stream general --num-before 100 | jq '.messages[] | select(.content | contains("/user_uploads/"))'
```

#### Get user group members

```bash
zulip-cli list-user-groups | jq '.user_groups[] | {name, member_count: (.members | length)}'
```

#### Filter active users only

```bash
zulip-cli list-users | jq '.members[] | select(.is_active == true) | {name: .full_name, email}'
```

### Scripting Examples

#### Send daily digest

```bash
#!/bin/bash
STREAM="general"
TOPIC="Daily Digest"
DATE=$(date +%Y-%m-%d)

# Get today's messages
MESSAGES=$(zulip-cli get-messages --stream "$STREAM" --num-before 100 | \
  jq -r ".messages[] | select(.timestamp > $(date -d 'today' +%s)) | \"- [\(.sender_full_name)]: \(.content)\""
)

# Send digest
zulip-cli send-message \
  --stream "$STREAM" \
  --topic "$TOPIC" \
  --content "**Digest for $DATE**\n\n$MESSAGES"
```

#### Monitor for mentions

```bash
#!/bin/bash
MY_NAME="Alice"

zulip-cli listen --messages-only | jq --unbuffered -r \
  "select(.sender_full_name != \"$MY_NAME\" and (.content | contains(\"@$MY_NAME\"))) | \
   \"[MENTION] \(.sender_full_name) in #\(.display_recipient)/\(.subject): \(.content)\""
```

#### Archive old messages

```bash
#!/bin/bash
# Get message IDs older than 30 days
CUTOFF=$(date -d '30 days ago' +%s)

zulip-cli get-messages --stream archive --num-before 1000 | \
  jq -r ".messages[] | select(.timestamp < $CUTOFF) | .id" | \
  while read -r msg_id; do
    echo "Deleting message $msg_id"
    zulip-cli delete-message "$msg_id"
  done
```

#### Bulk subscribe users to stream

```bash
#!/bin/bash
STREAM="announcements"

# Get all active user emails
USER_EMAILS=$(zulip-cli list-users | jq -r '.members[] | select(.is_active == true and .is_bot == false) | .email')

# Subscribe them all
for email in $USER_EMAILS; do
  echo "Subscribing $email to $STREAM"
  zulip-cli subscribe "$STREAM" --principals "$email"
done
```

#### Generate message statistics

```bash
#!/bin/bash
STREAM="general"

echo "Message Statistics for #$STREAM"
echo "================================"

MESSAGES=$(zulip-cli get-messages --stream "$STREAM" --num-before 500)

echo -n "Total messages: "
echo "$MESSAGES" | jq '.messages | length'

echo -n "Unique senders: "
echo "$MESSAGES" | jq '[.messages[].sender_full_name] | unique | length'

echo -n "Messages with reactions: "
echo "$MESSAGES" | jq '[.messages[] | select(.reactions | length > 0)] | length'

echo "Top 5 senders:"
echo "$MESSAGES" | jq -r '[.messages | group_by(.sender_full_name) | .[] | {sender: .[0].sender_full_name, count: length}] | sort_by(.count) | reverse | .[0:5] | .[] | "  \(.sender): \(.count) messages"'
```

#### Watch for keywords and notify

```bash
#!/bin/bash
KEYWORDS=("urgent" "critical" "help")

zulip-cli listen --messages-only | jq --unbuffered -r \
  "select(.content | ascii_downcase | test(\"$(IFS='|'; echo "${KEYWORDS[*]}")\")) | \
   \"[ALERT] \(.sender_full_name) in #\(.display_recipient)/\(.subject): \(.content)\"" | \
  while read -r alert; do
    echo "$alert"
    # Send notification (e.g., osascript, notify-send, etc.)
    osascript -e "display notification \"$alert\" with title \"Zulip Alert\""
  done
```

## Using as a Go Library

### Installation

```bash
go get github.com/intelligrit/zulip-cli
```

### Example Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/intelligrit/zulip-cli/client"
    "github.com/intelligrit/zulip-cli/types"
)

func main() {
    // Create client (uses ZULIP_URL, ZULIP_EMAIL, ZULIP_API_KEY env vars)
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

    // Listen for new messages
    err = c.CallOnEachMessage(func(msg types.Message) {
        fmt.Printf("New message from %s: %s\n", msg.SenderFullName, msg.Content)
    })
    if err != nil {
        log.Fatal(err)
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

## How It Works

zulip-cli is a complete Go reimplementation of the official Zulip Python API client. It provides:

- Full REST API coverage - every endpoint supported
- Type-safe request/response structures
- Automatic retry with exponential backoff
- Real-time event streaming via long-polling
- Efficient HTTP connection pooling
- Environment variable-based configuration

## Requirements

- Go 1.23+ (for building from source)
- A Zulip server (cloud or self-hosted)
- API credentials (email + API key)

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
./zulip-cli list-streams

# Run Go tests
go test ./...
```

## API Coverage

The library provides complete coverage of the Zulip API including:

- **Messages**: Send, fetch, update, delete, reactions, flags, rendering
- **Streams**: Create, update, delete, subscribe, topics, email addresses
- **Users**: List, create, update, deactivate, presence, alert words
- **User Groups**: Create, update, delete, manage members
- **Emoji**: List, upload, delete custom emoji
- **Realm**: Linkifiers, profile fields, server settings
- **Events**: Real-time event streaming and message listening
- **Files**: Upload and manage attachments

## Roadmap

Future enhancements:

- YAML output format support
- Table output format for better CLI readability
- Configuration file support (optional)
- Shell completion scripts
- Webhooks/outgoing webhooks support
- Message drafts management
- Typing indicators
- Read receipts

## Contributing

Contributions are welcome! This project follows standard Go conventions.

### Guidelines

1. Fork the repository
2. Create a feature branch
3. Make your changes following Go best practices
4. Write tests for new functionality
5. Ensure all tests pass
6. Run `go fmt` and `go vet`
7. Commit your changes
8. Push to the branch
9. Open a Pull Request

## About Intelligrit Labs

zulip-cli is developed by [Intelligrit Labs](https://intelligrit.com#labs), the R&D arm of Intelligrit LLC. We build tools for ourselves and release them for everyone. Intelligrit delivers AI-driven IT modernization for federal agencies.

## License

MIT License - see [LICENSE](LICENSE) file for details

## Support

For issues, questions, or contributions, please open an issue on GitHub.

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Complete reimplementation of the [python-zulip-api](https://github.com/zulip/python-zulip-api) library in Go
- Thanks to the Zulip team for their excellent API documentation
