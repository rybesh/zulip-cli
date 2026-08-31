package commands

import (
	"fmt"
	"strconv"

	"github.com/rybesh/zulip-cli/client"
	"github.com/spf13/cobra"
)

var listStreamsCmd = &cobra.Command{
	Use:     "list-channels",
	Aliases: []string{"list-streams"},
	Short:   "List all channels",
	RunE: func(cmd *cobra.Command, args []string) error {
		req := client.GetStreamsRequest{
			IncludePublic:     boolFlag(cmd, "include-public"),
			IncludeSubscribed: boolFlag(cmd, "include-subscribed"),
			IncludeAllActive:  boolFlag(cmd, "include-all-active"),
		}

		resp, err := zulipClient.GetStreams(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getStreamCmd = &cobra.Command{
	Use:     "get-channel [channel-name]",
	Aliases: []string{"get-stream"},
	Short:   "Get channel ID by name",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetStreamID(args[0])
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var createStreamCmd = &cobra.Command{
	Use:     "create-channel [name]",
	Aliases: []string{"create-stream"},
	Short:   "Create a new channel",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description, _ := cmd.Flags().GetString("description")

		req := client.CreateStreamRequest{
			Subscriptions: []struct {
				Name        string `json:"name"`
				Description string `json:"description,omitempty"`
			}{
				{Name: args[0], Description: description},
			},
			InviteOnly: boolFlag(cmd, "invite-only"),
			Announce:   boolFlag(cmd, "announce"),
		}

		resp, err := zulipClient.CreateStream(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var updateStreamCmd = &cobra.Command{
	Use:     "update-channel [channel-id]",
	Aliases: []string{"update-stream"},
	Short:   "Update a channel",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stream ID: %w", err)
		}

		description := stringFlag(cmd, "description")
		newName := stringFlag(cmd, "new-name")

		if description == nil && newName == nil {
			return fmt.Errorf("either --description or --new-name is required")
		}

		req := client.UpdateStreamRequest{
			StreamID:    streamID,
			Description: description,
			NewName:     newName,
		}

		resp, err := zulipClient.UpdateStream(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var deleteStreamCmd = &cobra.Command{
	Use:     "delete-channel [channel-id]",
	Aliases: []string{"delete-stream"},
	Short:   "Delete a channel",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stream ID: %w", err)
		}

		resp, err := zulipClient.DeleteStream(streamID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listStreamTopicsCmd = &cobra.Command{
	Use:     "list-channel-topics [channel-id]",
	Aliases: []string{"list-stream-topics"},
	Short:   "List all topics in a channel",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stream ID: %w", err)
		}

		resp, err := zulipClient.GetStreamTopics(streamID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var subscribeCmd = &cobra.Command{
	Use:   "subscribe [channel-names...]",
	Short: "Subscribe to one or more channels",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description, _ := cmd.Flags().GetString("description")

		var subscriptions []struct {
			Name        string `json:"name"`
			Description string `json:"description,omitempty"`
		}

		for _, name := range args {
			subscriptions = append(subscriptions, struct {
				Name        string `json:"name"`
				Description string `json:"description,omitempty"`
			}{Name: name, Description: description})
		}

		req := client.SubscribeRequest{
			Subscriptions: subscriptions,
		}

		resp, err := zulipClient.Subscribe(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var unsubscribeCmd = &cobra.Command{
	Use:   "unsubscribe [channel-names...]",
	Short: "Unsubscribe from one or more channels",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		req := client.UnsubscribeRequest{
			Subscriptions: args,
		}

		resp, err := zulipClient.Unsubscribe(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listSubscriptionsCmd = &cobra.Command{
	Use:   "list-subscriptions",
	Short: "List user's channel subscriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		req := client.GetSubscriptionsRequest{
			IncludeSubscribers: boolFlag(cmd, "include-subscribers"),
		}

		resp, err := zulipClient.GetSubscriptions(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listSubscribersCmd = &cobra.Command{
	Use:   "list-subscribers [channel-id]",
	Short: "List subscribers to a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stream ID: %w", err)
		}

		resp, err := zulipClient.GetSubscribers(streamID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var muteTopicCmd = &cobra.Command{
	Use:   "mute-topic [channel] [topic]",
	Short: "Mute a topic",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		req := client.MuteTopicRequest{
			Stream: args[0],
			Topic:  args[1],
			Op:     "add",
		}

		resp, err := zulipClient.MuteTopic(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var unmuteTopicCmd = &cobra.Command{
	Use:   "unmute-topic [channel] [topic]",
	Short: "Unmute a topic",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		req := client.MuteTopicRequest{
			Stream: args[0],
			Topic:  args[1],
			Op:     "remove",
		}

		resp, err := zulipClient.MuteTopic(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var moveTopicCmd = &cobra.Command{
	Use:   "move-topic [channel-id] [topic]",
	Short: "Move a topic to another channel or rename it",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stream ID: %w", err)
		}

		newChannelID, _ := cmd.Flags().GetInt("new-channel-id")
		newTopic := stringFlag(cmd, "new-topic")

		if newChannelID == 0 && newTopic == nil {
			return fmt.Errorf("either --new-channel-id or --new-topic is required")
		}

		req := client.MoveTopicRequest{
			StreamID:    streamID,
			Topic:       args[1],
			NewStreamID: newChannelID,
			NewTopic:    newTopic,
		}

		resp, err := zulipClient.MoveTopic(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

func init() {
	// Defaults match the server's, since an unset flag is left out of the
	// request; pass --include-public=false to actually exclude them.
	listStreamsCmd.Flags().Bool("include-public", true, "Include public channels")
	listStreamsCmd.Flags().Bool("include-subscribed", true, "Include subscribed channels")
	listStreamsCmd.Flags().Bool("include-all-active", false, "Include all active channels (admins only)")

	createStreamCmd.Flags().String("description", "", "Channel description")
	createStreamCmd.Flags().Bool("invite-only", false, "Make channel private")
	createStreamCmd.Flags().Bool("announce", false, "Announce channel creation")

	updateStreamCmd.Flags().String("description", "", "New description")
	updateStreamCmd.Flags().String("new-name", "", "New name")

	subscribeCmd.Flags().String("description", "", "Channel description (for new channels)")

	listSubscriptionsCmd.Flags().Bool("include-subscribers", false, "Include subscriber lists")

	moveTopicCmd.Flags().Int("new-channel-id", 0, "New channel ID (formerly --new-stream-id)")
	moveTopicCmd.Flags().String("new-topic", "", "New topic name")
}
