package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

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

	// newClient builds the client the commands use, and stdout is where their
	// results go. Both are variables so tests can run a command against a fake
	// server and read back what it printed.
	newClient           = client.NewClient
	stdout    io.Writer = os.Stdout

	// Global flags
	verbose bool
	output  string
	timeout time.Duration
)

// supportedOutputFormats lists the formats --output actually produces. Anything
// else is refused rather than quietly answered with JSON.
var supportedOutputFormats = []string{"json"}

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

Optional settings:
  - ZULIP_TIMEOUT: Request timeout, e.g. 45s (default 15s)
  - ZULIP_INSECURE: Set true to skip TLS verification (not recommended)
  - ZULIP_CERT_BUNDLE: PEM file of certificate authorities to trust
  - ZULIP_CLIENT_CERT, ZULIP_CLIENT_CERT_KEY: PEM client certificate and key

Zulip renamed streams to channels in version 9.0. This CLI follows that
naming; the older stream spellings still work as aliases.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Past this point failures are runtime failures, not usage errors, and
		// dumping the flag listing would only bury the message.
		cmd.SilenceUsage = true

		if !slices.Contains(supportedOutputFormats, output) {
			return fmt.Errorf("unsupported output format %q: supported formats are %s",
				output, strings.Join(supportedOutputFormats, ", "))
		}
		if timeout < 0 {
			return fmt.Errorf("--timeout must not be negative")
		}

		// These commands do not talk to a Zulip server, so they must not
		// require credentials.
		switch cmd.Name() {
		case "help", "completion", "version":
			return nil
		}

		var err error
		zulipClient, err = newClient()
		if err != nil {
			return fmt.Errorf("failed to initialize Zulip client: %w", err)
		}

		if verbose {
			zulipClient.Verbose = true
		}
		if timeout > 0 {
			zulipClient.Timeout = timeout
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
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "json", "Output format (json)")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 0,
		"Request timeout, e.g. 45s (default 15s, or $ZULIP_TIMEOUT)")

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
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printResult prints the result based on output format
func printResult(data interface{}) error {
	switch output {
	case "json":
		return printJSON(data)
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}

// boolFlag returns the flag's value only when the user set it, and nil
// otherwise. Request builders send only non-nil values, so an explicit
// --flag=false reaches the server instead of being dropped in favor of the
// server's own default.
func boolFlag(cmd *cobra.Command, name string) *bool {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		return nil
	}
	return &value
}

// intFlag is boolFlag for integers, so that a zero the user typed is told
// apart from a flag they never mentioned.
func intFlag(cmd *cobra.Command, name string) *int {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	value, err := cmd.Flags().GetInt(name)
	if err != nil {
		return nil
	}
	return &value
}

// stringFlag is boolFlag for strings, so that --description "" clears a value
// instead of meaning "leave it alone".
func stringFlag(cmd *cobra.Command, name string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return nil
	}
	return &value
}
