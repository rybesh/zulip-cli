package commands

import (
	"fmt"
	"strconv"

	"github.com/rybesh/zulip-cli/client"
	"github.com/spf13/cobra"
)

var listStreamsCmd = &cobra.Command{
	Use:   "list-streams",
	Short: "List all streams",
	RunE: func(cmd *cobra.Command, args []string) error {
		includePublic, _ := cmd.Flags().GetBool("include-public")
		includeSubscribed, _ := cmd.Flags().GetBool("include-subscribed")
		includeAllActive, _ := cmd.Flags().GetBool("include-all-active")

		req := client.GetStreamsRequest{
			IncludePublic:     includePublic,
			IncludeSubscribed: includeSubscribed,
			IncludeAllActive:  includeAllActive,
		}

		resp, err := zulipClient.GetStreams(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getStreamCmd = &cobra.Command{
	Use:   "get-stream [stream-name]",
	Short: "Get stream ID by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := zulipClient.GetStreamID(args[0])
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var createStreamCmd = &cobra.Command{
	Use:   "create-stream [name]",
	Short: "Create a new stream",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description, _ := cmd.Flags().GetString("description")
		inviteOnly, _ := cmd.Flags().GetBool("invite-only")
		announce, _ := cmd.Flags().GetBool("announce")

		req := client.CreateStreamRequest{
			Subscriptions: []struct {
				Name        string `json:"name"`
				Description string `json:"description,omitempty"`
			}{
				{Name: args[0], Description: description},
			},
			InviteOnly: inviteOnly,
			Announce:   announce,
		}

		resp, err := zulipClient.CreateStream(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var updateStreamCmd = &cobra.Command{
	Use:   "update-stream [stream-id]",
	Short: "Update a stream",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stream ID: %w", err)
		}

		description, _ := cmd.Flags().GetString("description")
		newName, _ := cmd.Flags().GetString("new-name")

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
	Use:   "delete-stream [stream-id]",
	Short: "Delete a stream",
	Args:  cobra.ExactArgs(1),
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
	Use:   "list-stream-topics [stream-id]",
	Short: "List all topics in a stream",
	Args:  cobra.ExactArgs(1),
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
	Use:   "subscribe [stream-names...]",
	Short: "Subscribe to one or more streams",
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
	Use:   "unsubscribe [stream-names...]",
	Short: "Unsubscribe from one or more streams",
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
	Short: "List user's stream subscriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		includeSubscribers, _ := cmd.Flags().GetBool("include-subscribers")

		req := client.GetSubscriptionsRequest{
			IncludeSubscribers: includeSubscribers,
		}

		resp, err := zulipClient.GetSubscriptions(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listSubscribersCmd = &cobra.Command{
	Use:   "list-subscribers [stream-id]",
	Short: "List subscribers to a stream",
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
	Use:   "mute-topic [stream] [topic]",
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
	Use:   "unmute-topic [stream] [topic]",
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
	Use:   "move-topic [stream-id] [topic]",
	Short: "Move a topic to another stream or rename it",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid stream ID: %w", err)
		}

		newStreamID, _ := cmd.Flags().GetInt("new-stream-id")
		newTopic, _ := cmd.Flags().GetString("new-topic")

		if newStreamID == 0 && newTopic == "" {
			return fmt.Errorf("either --new-stream-id or --new-topic is required")
		}

		req := client.MoveTopicRequest{
			StreamID:    streamID,
			Topic:       args[1],
			NewStreamID: newStreamID,
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
	listStreamsCmd.Flags().Bool("include-public", true, "Include public streams")
	listStreamsCmd.Flags().Bool("include-subscribed", false, "Include subscribed streams")
	listStreamsCmd.Flags().Bool("include-all-active", false, "Include all active streams")

	createStreamCmd.Flags().String("description", "", "Stream description")
	createStreamCmd.Flags().Bool("invite-only", false, "Make stream private")
	createStreamCmd.Flags().Bool("announce", false, "Announce stream creation")

	updateStreamCmd.Flags().String("description", "", "New description")
	updateStreamCmd.Flags().String("new-name", "", "New name")

	subscribeCmd.Flags().String("description", "", "Stream description (for new streams)")

	listSubscriptionsCmd.Flags().Bool("include-subscribers", false, "Include subscriber lists")

	moveTopicCmd.Flags().Int("new-stream-id", 0, "New stream ID")
	moveTopicCmd.Flags().String("new-topic", "", "New topic name")
}
