package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/intelligrit/zulip-cli/client"
	"github.com/intelligrit/zulip-cli/types"
	"github.com/spf13/cobra"
)

var sendMessageCmd = &cobra.Command{
	Use:   "send-message",
	Short: "Send a message",
	Long:  `Send a message to a stream or as a direct message`,
	RunE: func(cmd *cobra.Command, args []string) error {
		stream, _ := cmd.Flags().GetString("stream")
		topic, _ := cmd.Flags().GetString("topic")
		to, _ := cmd.Flags().GetStringSlice("to")
		content, _ := cmd.Flags().GetString("content")

		if content == "" {
			return fmt.Errorf("--content is required")
		}

		var req client.SendMessageRequest
		if stream != "" {
			if topic == "" {
				return fmt.Errorf("--topic is required for stream messages")
			}
			req = client.SendMessageRequest{
				Type:    "stream",
				To:      stream,
				Topic:   topic,
				Content: content,
			}
		} else if len(to) > 0 {
			req = client.SendMessageRequest{
				Type:    "private",
				To:      to,
				Content: content,
			}
		} else {
			return fmt.Errorf("either --stream or --to is required")
		}

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
	Long:  `Fetch messages with various filters`,
	RunE: func(cmd *cobra.Command, args []string) error {
		anchor, _ := cmd.Flags().GetString("anchor")
		numBefore, _ := cmd.Flags().GetInt("num-before")
		numAfter, _ := cmd.Flags().GetInt("num-after")
		stream, _ := cmd.Flags().GetString("stream")
		topic, _ := cmd.Flags().GetString("topic")

		req := client.GetMessagesRequest{
			Anchor:    anchor,
			NumBefore: numBefore,
			NumAfter:  numAfter,
		}

		// Build narrow
		var narrow []types.Narrow
		if stream != "" {
			narrow = append(narrow, types.Narrow{Operator: "stream", Operand: stream})
		}
		if topic != "" {
			narrow = append(narrow, types.Narrow{Operator: "topic", Operand: topic})
		}
		req.Narrow = narrow

		resp, err := zulipClient.GetMessages(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var updateMessageCmd = &cobra.Command{
	Use:   "update-message [message-id]",
	Short: "Update a message",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid message ID: %w", err)
		}

		content, _ := cmd.Flags().GetString("content")
		topic, _ := cmd.Flags().GetString("topic")

		if content == "" && topic == "" {
			return fmt.Errorf("either --content or --topic is required")
		}

		req := client.UpdateMessageRequest{
			MessageID: messageID,
			Content:   content,
			Topic:     topic,
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
	Use:   "add-reaction [message-id] [emoji-name]",
	Short: "Add a reaction to a message",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid message ID: %w", err)
		}

		req := client.AddReactionRequest{
			MessageID: messageID,
			EmojiName: args[1],
		}

		resp, err := zulipClient.AddReaction(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var removeReactionCmd = &cobra.Command{
	Use:   "remove-reaction [message-id] [emoji-name]",
	Short: "Remove a reaction from a message",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid message ID: %w", err)
		}

		req := client.AddReactionRequest{
			MessageID: messageID,
			EmojiName: args[1],
		}

		resp, err := zulipClient.RemoveReaction(req)
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
	Use:   "mark-stream-as-read [stream-id]",
	Short: "Mark all messages in a stream as read",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stream ID: %w", err)
		}

		resp, err := zulipClient.MarkStreamAsRead(streamID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var markTopicAsReadCmd = &cobra.Command{
	Use:   "mark-topic-as-read [stream-id] [topic]",
	Short: "Mark all messages in a topic as read",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stream ID: %w", err)
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
	sendMessageCmd.Flags().StringP("stream", "s", "", "Stream name")
	sendMessageCmd.Flags().StringP("topic", "t", "", "Topic name (required for stream messages)")
	sendMessageCmd.Flags().StringSliceP("to", "u", nil, "Recipients for direct message (comma-separated)")
	sendMessageCmd.Flags().StringP("content", "c", "", "Message content (required)")

	getMessagesCmd.Flags().String("anchor", "newest", "Anchor (newest, oldest, first_unread, or message ID)")
	getMessagesCmd.Flags().Int("num-before", 100, "Number of messages before anchor")
	getMessagesCmd.Flags().Int("num-after", 0, "Number of messages after anchor")
	getMessagesCmd.Flags().StringP("stream", "s", "", "Filter by stream")
	getMessagesCmd.Flags().StringP("topic", "t", "", "Filter by topic")

	updateMessageCmd.Flags().StringP("content", "c", "", "New message content")
	updateMessageCmd.Flags().StringP("topic", "t", "", "New topic name")
}
