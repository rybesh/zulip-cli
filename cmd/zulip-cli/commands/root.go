package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rybesh/zulip-cli/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// retiredFlagNames maps flag spellings this CLI no longer advertises to their
// current names. Zulip renamed streams to channels in version 9.0; the old
// spellings keep working so existing scripts do not break.
var retiredFlagNames = map[string]string{
	"stream":        "channel",
	"new-stream-id": "new-channel-id",
}

// normalizeFlagName lets a retired flag spelling resolve to its current flag.
// Unlike defining a second flag, this keeps one flag with one value, so
// --stream and --channel cannot disagree.
func normalizeFlagName(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if current, ok := retiredFlagNames[name]; ok {
		name = current
	}
	return pflag.NormalizedName(name)
}

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
manage channels, users, and more.

Authentication is done via environment variables:
  - ZULIP_URL: Your Zulip server URL
  - ZULIP_EMAIL: Your bot or user email
  - ZULIP_API_KEY: Your API key

Example:
  export ZULIP_URL=https://your-org.zulipchat.com
  export ZULIP_EMAIL=bot@example.com
  export ZULIP_API_KEY=your_api_key_here
  zulip-cli send-message --channel general --topic "Hello" --content "Hi there!"

Zulip renamed streams to channels in version 9.0. This CLI follows that
naming; the older stream spellings still work as aliases.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// These commands do not talk to a Zulip server, so they must not
		// require credentials or pay the cost of connecting.
		switch cmd.Name() {
		case "help", "completion", "version":
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
	// Setting Version makes cobra provide a --version flag. Cobra handles that
	// flag before PersistentPreRunE runs, so it needs no client either.
	rootCmd.Version = client.ClientVersion
	rootCmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")

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

	rootCmd.AddCommand(versionCmd)

	// Must follow AddCommand: this propagates to the commands registered above.
	rootCmd.SetGlobalNormalizationFunc(normalizeFlagName)
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
