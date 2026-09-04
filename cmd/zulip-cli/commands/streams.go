package commands

import (
	"fmt"
	"strconv"

	"github.com/rybesh/zulip-cli/client"
	"github.com/rybesh/zulip-cli/types"
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
		settings, err := channelSettings(cmd, &groupResolver{client: zulipClient})
		if err != nil {
			return err
		}

		req := client.CreateChannelRequest{
			Name:            args[0],
			Description:     stringFlag(cmd, "description"),
			Announce:        boolFlag(cmd, "announce"),
			ChannelSettings: settings,
		}

		if subscribers, _ := cmd.Flags().GetStringSlice("subscribers"); len(subscribers) > 0 {
			req.Subscribers, err = resolveUserIDs(zulipClient, subscribers)
			if err != nil {
				return err
			}
		}

		resp, err := zulipClient.CreateChannel(req)
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
		postingGroup := stringFlag(cmd, "can-send-message-group")

		if description == nil && newName == nil && postingGroup == nil {
			return fmt.Errorf("nothing to change: pass --description, --new-name, or --can-send-message-group")
		}

		req := client.UpdateStreamRequest{
			StreamID:    streamID,
			Description: description,
			NewName:     newName,
		}

		if postingGroup != nil {
			group, err := (&groupResolver{client: zulipClient}).resolve(*postingGroup)
			if err != nil {
				return fmt.Errorf("--can-send-message-group: %w", err)
			}
			req.CanSendMessageGroup = group
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

		var subscriptions []client.ChannelSubscription
		for _, name := range args {
			subscriptions = append(subscriptions, client.ChannelSubscription{
				Name: name, Description: description,
			})
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

// Muting, unmuting, following and unfollowing a topic are all one request
// with a different visibility policy, so these commands are the same command
// under four names people already reach for. Unmuting and unfollowing both
// clear the policy, which is what the server does for either.
var (
	muteTopicCmd = topicVisibilityCmd("mute-topic",
		"Mute a topic", types.VisibilityMuted)
	unmuteTopicCmd = topicVisibilityCmd("unmute-topic",
		"Clear a topic's visibility policy, unmuting it", types.VisibilityInherit)
	followTopicCmd = topicVisibilityCmd("follow-topic",
		"Follow a topic", types.VisibilityFollowed)
	unfollowTopicCmd = topicVisibilityCmd("unfollow-topic",
		"Clear a topic's visibility policy, unfollowing it", types.VisibilityInherit)
)

// topicVisibilityCmd builds a command that sets one fixed visibility policy
// for the topic named by its arguments.
func topicVisibilityCmd(name, short string, policy types.TopicVisibilityPolicy) *cobra.Command {
	return &cobra.Command{
		Use:   name + " [channel] [topic]",
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setTopicVisibility(args[0], args[1], policy)
		},
	}
}

var setTopicVisibilityCmd = &cobra.Command{
	Use:   "set-topic-visibility [channel] [topic] [policy]",
	Short: "Set your visibility policy for a topic",
	Long: `Set your own visibility policy for a topic.

The policy is one of:

  inherit   no policy of its own; the topic follows its channel (also "none")
  muted     hide the topic
  unmuted   show the topic even though its channel is muted
  followed  follow the topic

The mute-topic, unmute-topic, follow-topic and unfollow-topic commands are
shorthand for the policies they name.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		policy, err := types.ParseTopicVisibilityPolicy(args[2])
		if err != nil {
			return err
		}

		return setTopicVisibility(args[0], args[1], policy)
	},
}

// setTopicVisibility applies policy to a topic in the channel the user named,
// by ID or by name.
func setTopicVisibility(channel, topic string, policy types.TopicVisibilityPolicy) error {
	streamID, err := resolveChannelID(zulipClient, channel)
	if err != nil {
		return err
	}

	resp, err := zulipClient.UpdateUserTopic(client.UpdateUserTopicRequest{
		StreamID:         streamID,
		Topic:            topic,
		VisibilityPolicy: policy,
	})
	if err != nil {
		return err
	}

	return printResult(resp)
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
	createStreamCmd.Flags().Bool("announce", false, "Announce channel creation")
	createStreamCmd.Flags().StringSlice("subscribers", nil,
		"User IDs or email addresses to subscribe (default: just you)")
	addChannelSettingFlags(createStreamCmd)

	updateStreamCmd.Flags().String("description", "", "New description")
	updateStreamCmd.Flags().String("new-name", "", "New name")
	updateStreamCmd.Flags().String("can-send-message-group", "",
		"Who may post in the channel"+groupFlagHelp)

	subscribeCmd.Flags().String("description", "", "Channel description (for new channels)")

	listSubscriptionsCmd.Flags().Bool("include-subscribers", false, "Include subscriber lists")

	moveTopicCmd.Flags().Int("new-channel-id", 0, "New channel ID (formerly --new-stream-id)")
	moveTopicCmd.Flags().String("new-topic", "", "New topic name")
}

// channelPermissionFlags are the group-setting flags that configure who may do
// what in a channel. Each names the flag and the ChannelSettings field it
// fills; keeping them in one table is what lets create-channel offer the whole
// family without a dozen near-identical blocks.
var channelPermissionFlags = []struct {
	name  string
	help  string
	field func(*client.ChannelSettings) **types.GroupSetting
}{
	{"can-add-subscribers-group", "Who may add subscribers to the channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanAddSubscribersGroup }},
	{"can-administer-channel-group", "Who may administer the channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanAdministerChannelGroup }},
	{"can-create-topic-group", "Who may start a new topic in the channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanCreateTopicGroup }},
	{"can-delete-any-message-group", "Who may delete any message in the channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanDeleteAnyMessageGroup }},
	{"can-delete-own-message-group", "Who may delete their own messages in the channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanDeleteOwnMessageGroup }},
	{"can-move-messages-out-of-channel-group", "Who may move messages to another channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanMoveMessagesOutOfChannelGroup }},
	{"can-move-messages-within-channel-group", "Who may move messages between topics",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanMoveMessagesWithinChannelGroup }},
	{"can-remove-subscribers-group", "Who may remove subscribers from the channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanRemoveSubscribersGroup }},
	{"can-resolve-topics-group", "Who may resolve topics in the channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanResolveTopicsGroup }},
	{"can-send-message-group", "Who may post in the channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanSendMessageGroup }},
	{"can-subscribe-group", "Who may subscribe themselves to the channel",
		func(s *client.ChannelSettings) **types.GroupSetting { return &s.CanSubscribeGroup }},
}

// groupFlagHelp explains what a group-setting flag accepts. The role:* groups
// are the system groups every organization has.
const groupFlagHelp = " (user group ID or name, such as role:administrators)"

// addChannelSettingFlags registers the settings a channel can be created with.
func addChannelSettingFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("invite-only", false, "Make the channel private")
	cmd.Flags().Bool("is-web-public", false, "Make the channel readable by anyone on the internet")
	cmd.Flags().Bool("is-default-channel", false, "Subscribe new users to the channel automatically")
	cmd.Flags().Bool("history-public-to-subscribers", false, "Share history with users who subscribe later")
	cmd.Flags().Bool("default-push-notifications", false, "Enable mobile push notifications by default")
	cmd.Flags().String("message-retention-days", "",
		`Days to keep messages, or "realm_default" or "unlimited"`)
	cmd.Flags().Int("folder-id", 0, "Channel folder to file the channel under")
	cmd.Flags().String("topics-policy", "",
		`Topics allowed: inherit, allow_empty_topic, disable_empty_topic, or empty_topic_only`)

	for _, permission := range channelPermissionFlags {
		cmd.Flags().String(permission.name, "", permission.help+groupFlagHelp)
	}
}

// channelSettings reads the settings flags the user actually set. Group names
// are resolved against the server, so nothing is sent until every flag makes
// sense.
func channelSettings(cmd *cobra.Command, groups *groupResolver) (client.ChannelSettings, error) {
	settings := client.ChannelSettings{
		InviteOnly:                 boolFlag(cmd, "invite-only"),
		IsWebPublic:                boolFlag(cmd, "is-web-public"),
		IsDefaultStream:            boolFlag(cmd, "is-default-channel"),
		HistoryPublicToSubscribers: boolFlag(cmd, "history-public-to-subscribers"),
		DefaultPushNotifications:   boolFlag(cmd, "default-push-notifications"),
		FolderID:                   intFlag(cmd, "folder-id"),
		TopicsPolicy:               stringFlag(cmd, "topics-policy"),
	}

	if days := stringFlag(cmd, "message-retention-days"); days != nil {
		settings.MessageRetentionDays = client.RetentionDays(*days)
	}

	for _, permission := range channelPermissionFlags {
		value := stringFlag(cmd, permission.name)
		if value == nil {
			continue
		}
		group, err := groups.resolve(*value)
		if err != nil {
			return settings, fmt.Errorf("--%s: %w", permission.name, err)
		}
		*permission.field(&settings) = group
	}

	return settings, nil
}

// groupResolver turns what the user typed for a group-setting flag — a user
// group ID, or a group name such as role:administrators — into a value the
// server understands. Names cost one lookup, however many flags need them.
type groupResolver struct {
	client *client.Client
	byName map[string]int
}

func (r *groupResolver) resolve(value string) (*types.GroupSetting, error) {
	if id, err := strconv.Atoi(value); err == nil {
		return types.NamedGroup(id), nil
	}

	if r.byName == nil {
		resp, err := r.client.GetUserGroups()
		if err != nil {
			return nil, fmt.Errorf("failed to look up user group %q: %w", value, err)
		}
		r.byName = make(map[string]int, len(resp.UserGroups))
		for _, group := range resp.UserGroups {
			r.byName[group.Name] = group.ID
		}
	}

	id, found := r.byName[value]
	if !found {
		return nil, fmt.Errorf("no user group named %q; pass a group ID or a name from list-user-groups", value)
	}
	return types.NamedGroup(id), nil
}

// resolveChannelID turns what the user typed for a channel — a channel ID, or
// a channel name — into the ID the server wants.
func resolveChannelID(c *client.Client, value string) (int, error) {
	if id, err := strconv.Atoi(value); err == nil {
		return id, nil
	}

	resp, err := c.GetStreamID(value)
	if err != nil {
		return 0, fmt.Errorf("failed to look up channel %q: %w", value, err)
	}
	return resp.StreamID, nil
}

// resolveUserIDs turns user IDs and email addresses into the user IDs the
// server wants. Emails cost one lookup of the user list, however many there are.
func resolveUserIDs(c *client.Client, values []string) ([]int, error) {
	ids := make([]int, 0, len(values))
	var byEmail map[string]int

	for _, value := range values {
		if id, err := strconv.Atoi(value); err == nil {
			ids = append(ids, id)
			continue
		}

		if byEmail == nil {
			resp, err := c.GetUsers(client.GetUsersRequest{})
			if err != nil {
				return nil, fmt.Errorf("failed to look up user %q: %w", value, err)
			}
			byEmail = make(map[string]int, len(resp.Members))
			for _, user := range resp.Members {
				byEmail[user.Email] = user.UserID
				if user.DeliveryEmail != "" {
					byEmail[user.DeliveryEmail] = user.UserID
				}
			}
		}

		id, found := byEmail[value]
		if !found {
			return nil, fmt.Errorf("no user with email %q; pass a user ID or an address from list-users", value)
		}
		ids = append(ids, id)
	}

	return ids, nil
}
