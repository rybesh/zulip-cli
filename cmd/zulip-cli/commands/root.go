package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/intelligrit/zulip-cli/client"
	"github.com/spf13/cobra"
)

var (
	// Global client instance
	zulipClient *client.Client

	// Global flags
	verbose bool
	output  string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "zulip-cli",
	Short: "A comprehensive CLI for Zulip",
	Long: `zulip-cli is a command-line interface for interacting with Zulip.
It provides full access to the Zulip API, allowing you to send messages,
manage streams, users, and more.

Authentication is done via environment variables:
  - ZULIP_URL: Your Zulip server URL
  - ZULIP_EMAIL: Your bot or user email
  - ZULIP_API_KEY: Your API key

Example:
  export ZULIP_URL=https://your-org.zulipchat.com
  export ZULIP_EMAIL=bot@example.com
  export ZULIP_API_KEY=your_api_key_here
  zulip-cli send-message --stream general --topic "Hello" --content "Hi there!"`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip client initialization for help and version commands
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}

		var err error
		zulipClient, err = client.NewClient()
		if err != nil {
			return fmt.Errorf("failed to initialize Zulip client: %w", err)
		}

		if verbose {
			zulipClient.Verbose = true
		}

		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "json", "Output format (json, yaml, table)")

	// Add all subcommands
	rootCmd.AddCommand(sendMessageCmd)
	rootCmd.AddCommand(getMessagesCmd)
	rootCmd.AddCommand(updateMessageCmd)
	rootCmd.AddCommand(deleteMessageCmd)
	rootCmd.AddCommand(addReactionCmd)
	rootCmd.AddCommand(removeReactionCmd)
	rootCmd.AddCommand(uploadFileCmd)
	rootCmd.AddCommand(markAllAsReadCmd)
	rootCmd.AddCommand(markStreamAsReadCmd)
	rootCmd.AddCommand(markTopicAsReadCmd)
	rootCmd.AddCommand(getMessageHistoryCmd)

	rootCmd.AddCommand(listStreamsCmd)
	rootCmd.AddCommand(getStreamCmd)
	rootCmd.AddCommand(createStreamCmd)
	rootCmd.AddCommand(updateStreamCmd)
	rootCmd.AddCommand(deleteStreamCmd)
	rootCmd.AddCommand(listStreamTopicsCmd)
	rootCmd.AddCommand(subscribeCmd)
	rootCmd.AddCommand(unsubscribeCmd)
	rootCmd.AddCommand(listSubscriptionsCmd)
	rootCmd.AddCommand(listSubscribersCmd)
	rootCmd.AddCommand(muteTopicCmd)
	rootCmd.AddCommand(unmuteTopicCmd)
	rootCmd.AddCommand(moveTopicCmd)

	rootCmd.AddCommand(listUsersCmd)
	rootCmd.AddCommand(getUserCmd)
	rootCmd.AddCommand(getProfileCmd)
	rootCmd.AddCommand(createUserCmd)
	rootCmd.AddCommand(updateUserCmd)
	rootCmd.AddCommand(deactivateUserCmd)
	rootCmd.AddCommand(reactivateUserCmd)
	rootCmd.AddCommand(getUserPresenceCmd)
	rootCmd.AddCommand(updatePresenceCmd)

	rootCmd.AddCommand(listUserGroupsCmd)
	rootCmd.AddCommand(createUserGroupCmd)
	rootCmd.AddCommand(updateUserGroupCmd)
	rootCmd.AddCommand(deleteUserGroupCmd)
	rootCmd.AddCommand(addGroupMembersCmd)
	rootCmd.AddCommand(removeGroupMembersCmd)

	rootCmd.AddCommand(listEmojiCmd)
	rootCmd.AddCommand(uploadEmojiCmd)
	rootCmd.AddCommand(deleteEmojiCmd)

	rootCmd.AddCommand(listAlertWordsCmd)
	rootCmd.AddCommand(addAlertWordsCmd)
	rootCmd.AddCommand(removeAlertWordsCmd)

	rootCmd.AddCommand(serverSettingsCmd)
	rootCmd.AddCommand(listenCmd)
}

// printJSON prints data as JSON
func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printResult prints the result based on output format
func printResult(data interface{}) error {
	switch output {
	case "json":
		return printJSON(data)
	default:
		return printJSON(data)
	}
}
