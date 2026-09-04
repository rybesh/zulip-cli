package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rybesh/zulip-cli/client"
	"github.com/rybesh/zulip-cli/types"
	"github.com/spf13/cobra"
)

// directRecipients turns what the user typed after --to into the form the
// server reads. A list of numbers is a list of user IDs, and has to travel as
// numbers: sent as strings the server reads them as email addresses and
// rejects them. Anything else is a list of email addresses, which travels
// unchanged. The two cannot be mixed, so a single unparseable entry keeps the
// whole list as addresses.
func directRecipients(to []string) interface{} {
	ids := make([]int, 0, len(to))
	for _, recipient := range to {
		id, err := strconv.Atoi(recipient)
		if err != nil {
			return to
		}
		ids = append(ids, id)
	}
	return ids
}

var sendMessageCmd = &cobra.Command{
	Use:   "send-message",
	Short: "Send a message",
	Long:  `Send a message to a channel or as a direct message`,
	RunE: func(cmd *cobra.Command, args []string) error {
		channel, _ := cmd.Flags().GetString("channel")
		topic, _ := cmd.Flags().GetString("topic")
		to, _ := cmd.Flags().GetStringSlice("to")
		content, _ := cmd.Flags().GetString("content")
		queueID, _ := cmd.Flags().GetString("queue-id")
		localID, _ := cmd.Flags().GetString("local-id")

		if content == "" {
			return fmt.Errorf("--content is required")
		}
		if localID != "" && queueID == "" {
			return fmt.Errorf("--local-id needs --queue-id: the server reads the two together")
		}

		var req client.SendMessageRequest
		if channel != "" {
			if topic == "" {
				return fmt.Errorf("--topic is required for channel messages")
			}
			req = client.SendMessageRequest{
				// "stream" is the wire value; the server has not renamed it.
				Type:    "stream",
				To:      channel,
				Topic:   topic,
				Content: content,
			}
		} else if len(to) > 0 {
			req = client.SendMessageRequest{
				Type:    "private",
				To:      directRecipients(to),
				Content: content,
			}
		} else {
			return fmt.Errorf("either --channel or --to is required")
		}

		req.QueueID = queueID
		req.LocalID = localID

		resp, err := zulipClient.SendMessage(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getMessagesCmd = &cobra.Command{
	Use:   "get-messages",
	Short: "Fetch messages",
	Long: `Fetch messages with various filters.

--channel and --topic are shorthand for the two narrow operators people reach
for most. --narrow takes any operator the server understands, and may be
repeated; every filter given is combined:

  zulip-cli get-messages --narrow sender=me@example.com --narrow is=unread
  zulip-cli get-messages --channel general --narrow has=link
  zulip-cli get-messages --narrow search="release notes"

Message content arrives as rendered HTML unless --no-markdown is given, which
asks for the Markdown the sender typed. To walk further back than one request
returns, --all keeps fetching older messages until the history runs out.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		anchor, _ := cmd.Flags().GetString("anchor")
		numBefore, _ := cmd.Flags().GetInt("num-before")
		numAfter, _ := cmd.Flags().GetInt("num-after")
		all, _ := cmd.Flags().GetBool("all")

		if all && numAfter > 0 {
			return fmt.Errorf("--all walks backwards through the history, so it cannot be combined with --num-after")
		}

		narrow, err := narrowFlag(cmd)
		if err != nil {
			return err
		}

		req := client.GetMessagesRequest{
			Anchor:               anchor,
			NumBefore:            numBefore,
			NumAfter:             numAfter,
			Narrow:               narrow,
			ClientGravatar:       boolFlag(cmd, "client-gravatar"),
			UseFirstUnreadAnchor: boolFlag(cmd, "use-first-unread-anchor"),
		}

		// --no-markdown is the negative of the parameter the server takes, so
		// that the common case reads as the thing being asked for.
		if noMarkdown := boolFlag(cmd, "no-markdown"); noMarkdown != nil {
			applyMarkdown := !*noMarkdown
			req.ApplyMarkdown = &applyMarkdown
		}

		var resp *client.GetMessagesResponse
		if all {
			resp, err = fetchAllMessages(req)
		} else {
			resp, err = zulipClient.GetMessages(req)
		}
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

// fetchAllMessages walks backwards from the anchor a page at a time, until the
// server says it has reached the oldest message the request can see. Each page
// re-anchors just below the oldest message of the one before, so no message is
// reported twice. The pages are merged into the single response a caller would
// have got if one request could have returned everything.
func fetchAllMessages(req client.GetMessagesRequest) (*client.GetMessagesResponse, error) {
	var all *client.GetMessagesResponse

	for {
		page, err := zulipClient.GetMessages(req)
		if err != nil {
			return nil, err
		}

		if all == nil {
			all = page
		} else {
			// Pages arrive oldest first, and each is older than the last.
			all.Messages = append(page.Messages, all.Messages...)
			all.FoundOldest = page.FoundOldest
			all.HistoryLimited = all.HistoryLimited || page.HistoryLimited
		}

		if all.FoundOldest || len(page.Messages) == 0 {
			return all, nil
		}

		// Anchoring one below the oldest message asks for messages strictly
		// older than the ones already in hand.
		req.Anchor = page.Messages[0].ID - 1
		// A first-unread anchor would override the one just computed.
		req.UseFirstUnreadAnchor = nil
	}
}

var getRawMessageCmd = &cobra.Command{
	Use:     "get-raw-message [message-id]",
	Aliases: []string{"get-message-markdown"},
	Short:   "Get a message's original Markdown",
	Long: `Get the Markdown a message was written in, rather than the HTML the
server renders it to. get-messages --no-markdown does the same for many
messages at once.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid message ID: %w", err)
		}

		resp, err := zulipClient.GetRawMessage(messageID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var updateMessageCmd = &cobra.Command{
	Use:   "update-message [message-id]",
	Short: "Update a message",
	Long: `Edit a message's content, its topic, or which channel it is in.

A topic or channel edit moves only the message named, unless --propagate-mode
says otherwise:

  change_one    just this message (the server's default)
  change_later  this message and the ones after it in the topic
  change_all    every message in the topic

Renaming a whole thread means --propagate-mode change_all.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid message ID: %w", err)
		}

		content := stringFlag(cmd, "content")
		topic := stringFlag(cmd, "topic")
		channelID, _ := cmd.Flags().GetInt("channel-id")

		if content == nil && topic == nil && channelID == 0 {
			return fmt.Errorf("either --content, --topic or --channel-id is required")
		}

		req := client.UpdateMessageRequest{
			MessageID:                   messageID,
			Content:                     content,
			Topic:                       topic,
			StreamID:                    channelID,
			SendNotificationToOldThread: boolFlag(cmd, "send-notification-to-old-thread"),
			SendNotificationToNewThread: boolFlag(cmd, "send-notification-to-new-thread"),
		}

		if mode := stringFlag(cmd, "propagate-mode"); mode != nil {
			req.PropagateMode, err = types.ParseEditPropagateMode(*mode)
			if err != nil {
				return err
			}
		}

		resp, err := zulipClient.UpdateMessage(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var deleteMessageCmd = &cobra.Command{
	Use:   "delete-message [message-id]",
	Short: "Delete a message",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid message ID: %w", err)
		}

		resp, err := zulipClient.DeleteMessage(messageID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var addReactionCmd = &cobra.Command{
	Use:     "add-reaction [message-id] [emoji-name]",
	Short:   "Add a reaction to a message",
	Long:    reactionHelp("Add a reaction to a message."),
	Args:    cobra.RangeArgs(1, 2),
	PreRunE: reactionArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		req, err := reactionRequest(cmd, args)
		if err != nil {
			return err
		}

		resp, err := zulipClient.AddReaction(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var removeReactionCmd = &cobra.Command{
	Use:     "remove-reaction [message-id] [emoji-name]",
	Short:   "Remove a reaction from a message",
	Long:    reactionHelp("Remove one of your reactions from a message."),
	Args:    cobra.RangeArgs(1, 2),
	PreRunE: reactionArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		req, err := reactionRequest(cmd, args)
		if err != nil {
			return err
		}

		resp, err := zulipClient.RemoveReaction(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

// reactionHelp describes the ways of naming an emoji, which both reaction
// commands accept in the same way.
func reactionHelp(what string) string {
	return what + `

The emoji is always named, including custom emoji added to the organization.
--emoji-code and --reaction-type say which emoji a name belongs to, for the
rare name that more than one of them answers to:

  zulip-cli add-reaction 42 tada
  zulip-cli add-reaction 42 tada --emoji-code 1f389 --reaction-type unicode_emoji
  zulip-cli add-reaction 42 party-parrot --emoji-code 7 --reaction-type realm_emoji`
}

// reactionArgs refuses a reaction that names no emoji, before the command
// builds a request the server would only reject: the server wants the name
// whatever else it is given.
func reactionArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("name the emoji: the server needs its name even alongside --emoji-code")
	}
	return nil
}

// reactionRequest builds the request both reaction commands send.
func reactionRequest(cmd *cobra.Command, args []string) (client.AddReactionRequest, error) {
	var req client.AddReactionRequest

	messageID, err := strconv.Atoi(args[0])
	if err != nil {
		return req, fmt.Errorf("invalid message ID: %w", err)
	}
	req.MessageID = messageID

	if len(args) > 1 {
		req.EmojiName = args[1]
	}
	req.EmojiCode, _ = cmd.Flags().GetString("emoji-code")

	if reactionType := stringFlag(cmd, "reaction-type"); reactionType != nil {
		req.ReactionType, err = types.ParseEmojiType(*reactionType)
		if err != nil {
			return req, err
		}
	}

	return req, nil
}

var updateMessageFlagsCmd = &cobra.Command{
	Use:   "update-message-flags [add|remove] [flag] [message-ids...]",
	Short: "Add or remove a flag on specific messages",
	Long: `Add or remove one of your own flags on the messages named.

The flag is one of:

  read       whether you have read the message
  starred    whether you have starred it
  collapsed  whether it is collapsed in your view

The other flags a message carries — mentioned, has_alert_word and the rest —
are the server's to decide, and it refuses to be told what they are.

  zulip-cli update-message-flags add read 41 42 43
  zulip-cli update-message-flags remove read 41
  zulip-cli update-message-flags add starred 42`,
	Args: cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		op := strings.ToLower(args[0])
		if op != "add" && op != "remove" {
			return fmt.Errorf("unknown operation %q: it must be add or remove", args[0])
		}

		flag, err := types.ParseMessageFlag(args[1])
		if err != nil {
			return err
		}

		messages := make([]int, 0, len(args)-2)
		for _, arg := range args[2:] {
			id, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("invalid message ID %q: %w", arg, err)
			}
			messages = append(messages, id)
		}

		resp, err := zulipClient.UpdateMessageFlags(client.UpdateMessageFlagsRequest{
			Messages: messages,
			Op:       op,
			Flag:     flag,
		})
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var renderMessageCmd = &cobra.Command{
	Use:   "render-message [content]",
	Short: "Render Markdown the way the server would",
	Long: `Render message Markdown to HTML without sending anything.

This is how to check what a message will look like before sending it.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.RenderMessage(args[0])
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var checkMessagesMatchNarrowCmd = &cobra.Command{
	Use:   "check-messages-match-narrow [message-ids...]",
	Short: "Check which of the messages match a narrow",
	Long: `Check which of the messages named match the narrow given.

--narrow takes the same operator=operand filters as get-messages, and may be
repeated. Nothing is fetched: the answer says only which messages matched.

  zulip-cli check-messages-match-narrow 41 42 --narrow has=link`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		narrow, err := narrowFlag(cmd)
		if err != nil {
			return err
		}
		if len(narrow) == 0 {
			return fmt.Errorf("--narrow is required: there is nothing to match against without it")
		}

		messageIDs := make([]int, 0, len(args))
		for _, arg := range args {
			id, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("invalid message ID %q: %w", arg, err)
			}
			messageIDs = append(messageIDs, id)
		}

		resp, err := zulipClient.CheckMessagesMatchNarrow(client.CheckMessagesMatchNarrowRequest{
			MessageIDs: messageIDs,
			Narrow:     narrow,
		})
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var uploadFileCmd = &cobra.Command{
	Use:   "upload-file [filename]",
	Short: "Upload a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]
		file, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		resp, err := zulipClient.UploadFile(file, filename)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listAttachmentsCmd = &cobra.Command{
	Use:   "list-attachments",
	Short: "List the files you have uploaded",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetAttachments()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var markAllAsReadCmd = &cobra.Command{
	Use:   "mark-all-as-read",
	Short: "Mark all messages as read",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.MarkAllAsRead()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var markStreamAsReadCmd = &cobra.Command{
	Use:     "mark-channel-as-read [channel]",
	Aliases: []string{"mark-stream-as-read"},
	Short:   "Mark all messages in a channel as read",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
		}

		resp, err := zulipClient.MarkStreamAsRead(streamID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var markTopicAsReadCmd = &cobra.Command{
	Use:   "mark-topic-as-read [channel] [topic]",
	Short: "Mark all messages in a topic as read",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
		}

		resp, err := zulipClient.MarkTopicAsRead(streamID, args[1])
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getMessageHistoryCmd = &cobra.Command{
	Use:   "get-message-history [message-id]",
	Short: "Get the edit history of a message",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid message ID: %w", err)
		}

		resp, err := zulipClient.GetMessageHistory(messageID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

func init() {
	sendMessageCmd.Flags().StringP("channel", "s", "", "Channel name (formerly stream)")
	sendMessageCmd.Flags().StringP("topic", "t", "", "Topic name (required for channel messages)")
	sendMessageCmd.Flags().StringSliceP("to", "u", nil, "Recipients for direct message (comma-separated)")
	sendMessageCmd.Flags().StringP("content", "c", "", "Message content (required)")
	sendMessageCmd.Flags().String("queue-id", "",
		"Event queue that should recognize the message as its own")
	sendMessageCmd.Flags().String("local-id", "",
		"Your own identifier for the message, echoed back on the queue named by --queue-id")

	getMessagesCmd.Flags().String("anchor", "newest", "Anchor (newest, oldest, first_unread, or message ID)")
	getMessagesCmd.Flags().Int("num-before", 100, "Number of messages before anchor")
	getMessagesCmd.Flags().Int("num-after", 0, "Number of messages after anchor")
	getMessagesCmd.Flags().StringP("channel", "s", "", "Filter by channel (formerly stream)")
	getMessagesCmd.Flags().StringP("topic", "t", "", "Filter by topic")
	getMessagesCmd.Flags().Bool("no-markdown", false,
		"Return the Markdown the sender typed instead of rendered HTML")
	getMessagesCmd.Flags().Bool("client-gravatar", false,
		"Leave out avatar URLs that can be computed from the sender's email")
	getMessagesCmd.Flags().Bool("use-first-unread-anchor", false,
		"Anchor on your first unread message, ignoring --anchor")
	getMessagesCmd.Flags().Bool("all", false,
		"Keep fetching older messages, --num-before at a time, until the history runs out")
	addNarrowFlag(getMessagesCmd)

	addNarrowFlag(checkMessagesMatchNarrowCmd)

	updateMessageCmd.Flags().StringP("content", "c", "", "New message content")
	updateMessageCmd.Flags().StringP("topic", "t", "", "New topic name")
	updateMessageCmd.Flags().Int("channel-id", 0, "Move the message to this channel")
	updateMessageCmd.Flags().String("propagate-mode", "",
		"Which messages a topic or channel edit moves: change_one, change_later, or change_all")
	updateMessageCmd.Flags().Bool("send-notification-to-old-thread", false,
		"Leave a notice in the thread the message came from")
	updateMessageCmd.Flags().Bool("send-notification-to-new-thread", false,
		"Leave a notice in the thread the message moved to")

	for _, cmd := range []*cobra.Command{addReactionCmd, removeReactionCmd} {
		cmd.Flags().String("emoji-code", "",
			"Emoji code, for custom emoji and for naming a Unicode emoji by code point")
		cmd.Flags().String("reaction-type", "",
			"Kind of emoji --emoji-code names: unicode_emoji, realm_emoji, or zulip_extra_emoji")
	}
}

// addNarrowFlag registers the repeatable --narrow filter. It is a StringArray
// rather than a StringSlice because operands routinely contain commas, and a
// search phrase should not be split into two filters.
func addNarrowFlag(cmd *cobra.Command) {
	cmd.Flags().StringArray("narrow", nil,
		"Filter as operator=operand, repeatable (e.g. sender=me@example.com, is=unread, has=link)")
}

// narrowFlag reads the narrow the user asked for, starting with the --channel
// and --topic shorthands when the command offers them. Every operator the
// server understands is reachable through --narrow, so a filter this CLI has
// never heard of still works.
func narrowFlag(cmd *cobra.Command) ([]types.Narrow, error) {
	var narrow []types.Narrow

	if channel, err := cmd.Flags().GetString("channel"); err == nil && channel != "" {
		// "stream" is the wire operator; the server has not renamed it.
		narrow = append(narrow, types.Narrow{Operator: "stream", Operand: channel})
	}
	if topic, err := cmd.Flags().GetString("topic"); err == nil && topic != "" {
		narrow = append(narrow, types.Narrow{Operator: "topic", Operand: topic})
	}

	values, _ := cmd.Flags().GetStringArray("narrow")
	for _, value := range values {
		operator, operand, found := strings.Cut(value, "=")
		if !found || operator == "" {
			return nil, fmt.Errorf(
				"invalid --narrow %q: it must be operator=operand, such as sender=me@example.com", value)
		}
		narrow = append(narrow, types.Narrow{Operator: operator, Operand: operand})
	}

	return narrow, nil
}
