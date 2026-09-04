package commands

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rybesh/zulip-cli/client"
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

		resp, err := zulipClient.UploadCustomEmoji(args[0], args[1], file)
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
With no --event-types, listens for messages only.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eventTypes, _ := cmd.Flags().GetStringSlice("event-types")
		messagesOnly, _ := cmd.Flags().GetBool("messages-only")

		if messagesOnly && len(eventTypes) > 0 {
			return fmt.Errorf("--messages-only and --event-types cannot be combined")
		}

		// Ctrl-C ends the listener through the context, which lets it release
		// its event queue on the server instead of abandoning it.
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		// Status goes to stderr so that piping the events into jq keeps working.
		if messagesOnly || len(eventTypes) == 0 {
			fmt.Fprintln(os.Stderr, "Listening for messages... (Press Ctrl+C to stop)")
			// Each message is printed as JSON, one after another, for the same
			// reason every other command prints JSON: so that stdout can be
			// piped straight into jq.
			return zulipClient.CallOnEachMessage(ctx, func(msg types.Message) {
				if err := printJSON(msg); err != nil {
					fmt.Fprintf(os.Stderr, "failed to print message %d: %v\n", msg.ID, err)
				}
			})
		}

		fmt.Fprintf(os.Stderr, "Listening for events: %s... (Press Ctrl+C to stop)\n", strings.Join(eventTypes, ", "))
		return zulipClient.CallOnEachEvent(ctx, func(event map[string]interface{}) {
			printJSON(event)
		}, eventTypes, nil)
	},
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register an event queue",
	Long: `Register an event queue and print its ID.

listen keeps a queue of its own and needs none of this. Registering one by
hand is for driving the queue from a script: hold the queue ID and the last
event ID it reports, poll with get-events, and release it with deregister when
the script is done.

With no --event-types the server sends every event type it has for you.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eventTypes, _ := cmd.Flags().GetStringSlice("event-types")

		resp, err := zulipClient.Register(client.RegisterRequest{
			EventTypes:         eventTypes,
			AllPublicStreams:   boolFlag(cmd, "all-public-channels"),
			IncludeSubscribers: boolFlag(cmd, "include-subscribers"),
			ClientGravatar:     boolFlag(cmd, "client-gravatar"),
			SlimPresence:       boolFlag(cmd, "slim-presence"),
			ApplyMarkdown:      boolFlag(cmd, "apply-markdown"),
		})
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getEventsCmd = &cobra.Command{
	Use:   "get-events [queue-id]",
	Short: "Fetch events from a queue",
	Long: `Fetch the events a queue has collected since a given event ID.

The queue comes from register. --last-event-id is where to read from: the
value register reported the first time, and afterwards the largest id among
the events of the previous fetch, so that no event is read twice.

The server holds the request open until it has something to say. --dont-block
takes whatever has already arrived and returns at once.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		lastEventID, _ := cmd.Flags().GetInt("last-event-id")
		dontBlock, _ := cmd.Flags().GetBool("dont-block")

		resp, err := zulipClient.GetEvents(client.GetEventsRequest{
			QueueID:     args[0],
			LastEventID: lastEventID,
			DontBlock:   dontBlock,
		})
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var deregisterCmd = &cobra.Command{
	Use:   "deregister [queue-id]",
	Short: "Release an event queue",
	Long: `Release an event queue the server is still holding open.

The listen command releases its own queue when it is stopped with Ctrl-C, but
a queue survives a listener that was killed outright, and the server goes on
collecting events for it until it expires. This releases one early.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.Deregister(args[0])
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

func init() {
	listenCmd.Flags().StringSlice("event-types", nil, "Event types to listen for (comma-separated)")
	listenCmd.Flags().Bool("messages-only", false, "Listen for messages only (the default when no --event-types are given)")

	registerCmd.Flags().StringSlice("event-types", nil, "Event types the queue should collect (comma-separated); every type by default")
	registerCmd.Flags().Bool("all-public-channels", false, "Collect messages from every public channel, not only the ones you subscribe to")
	registerCmd.Flags().Bool("include-subscribers", false, "Include each channel's subscribers in the initial state")
	registerCmd.Flags().Bool("client-gravatar", false, "Leave out avatar URLs that can be computed from the sender's email")
	registerCmd.Flags().Bool("slim-presence", false, "Report presence by user ID rather than by email address")
	registerCmd.Flags().Bool("apply-markdown", true, "Render message content to HTML, rather than the Markdown the sender typed")

	getEventsCmd.Flags().Int("last-event-id", -1, "Fetch events newer than this ID")
	getEventsCmd.Flags().Bool("dont-block", false, "Return whatever has already arrived instead of waiting for an event")
}
