package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/rybesh/zulip-cli/types"
	"github.com/spf13/cobra"
)

var serverSettingsCmd = &cobra.Command{
	Use:   "server-settings",
	Short: "Get server settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetServerSettings()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listEmojiCmd = &cobra.Command{
	Use:   "list-emoji",
	Short: "List all custom emoji",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetRealmEmoji()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var uploadEmojiCmd = &cobra.Command{
	Use:   "upload-emoji [name] [file]",
	Short: "Upload a custom emoji",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		file, err := os.Open(args[1])
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		resp, err := zulipClient.UploadCustomEmoji(args[0], file)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var deleteEmojiCmd = &cobra.Command{
	Use:   "delete-emoji [name]",
	Short: "Delete a custom emoji",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.DeleteCustomEmoji(args[0])
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listAlertWordsCmd = &cobra.Command{
	Use:   "list-alert-words",
	Short: "List user's alert words",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetAlertWords()
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var addAlertWordsCmd = &cobra.Command{
	Use:   "add-alert-words [words...]",
	Short: "Add alert words",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.AddAlertWords(args)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var removeAlertWordsCmd = &cobra.Command{
	Use:   "remove-alert-words [words...]",
	Short: "Remove alert words",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.RemoveAlertWords(args)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Listen for events and messages",
	Long: `Listen for events and messages from Zulip.
By default, listens for all message events.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eventTypes, _ := cmd.Flags().GetStringSlice("event-types")
		messagesOnly, _ := cmd.Flags().GetBool("messages-only")

		if messagesOnly || len(eventTypes) == 0 {
			fmt.Println("Listening for messages... (Press Ctrl+C to stop)")
			return zulipClient.CallOnEachMessage(func(msg types.Message) {
				fmt.Printf("\n[%s] %s: %s\n", msg.Type, msg.SenderFullName, msg.Content)
				if msg.Type == "stream" {
					fmt.Printf("  Channel: %v | Topic: %s\n", msg.DisplayRecipient, msg.Subject)
				}
			})
		}

		fmt.Printf("Listening for events: %s... (Press Ctrl+C to stop)\n", strings.Join(eventTypes, ", "))
		return zulipClient.CallOnEachEvent(func(event map[string]interface{}) {
			printJSON(event)
		}, eventTypes, nil)
	},
}

func init() {
	listenCmd.Flags().StringSlice("event-types", nil, "Event types to listen for (comma-separated)")
	listenCmd.Flags().Bool("messages-only", true, "Listen for messages only")
}
