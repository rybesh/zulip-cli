package commands

import (
	"fmt"
	"strconv"
	"strings"

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
	Use:     "get-channel [channel]",
	Aliases: []string{"get-stream"},
	Short:   "Get a channel by ID or name",
	Long: `Get everything the server knows about one channel.

get-channel-id is the way to ask only for a channel's ID.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
		}

		resp, err := zulipClient.GetStream(streamID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getStreamIDCmd = &cobra.Command{
	Use:     "get-channel-id [channel-name]",
	Aliases: []string{"get-stream-id"},
	Short:   "Get a channel's ID by name",
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

// updateChannelFlags are the settings update-channel can change. A request
// that names none of them is a mistake worth catching before it is sent.
var updateChannelFlags = []string{
	"description", "new-name", "invite-only", "is-web-public",
	"history-public-to-subscribers", "message-retention-days",
	"can-send-message-group",
}

var updateStreamCmd = &cobra.Command{
	Use:     "update-channel [channel]",
	Aliases: []string{"update-stream"},
	Short:   "Update a channel",
	Long: `Update a channel's name, description, privacy, or retention policy.

The three privacy settings are tri-state. Leaving one out changes nothing, and
--invite-only=false makes a private channel public rather than being taken as
"leave it alone".

Making a public channel private, or the other way around, may also need
--history-public-to-subscribers: the server decides what happens to the
existing history from the two together.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
		}

		changed := false
		for _, name := range updateChannelFlags {
			if cmd.Flags().Changed(name) {
				changed = true
				break
			}
		}
		if !changed {
			return fmt.Errorf("nothing to change: pass --%s", strings.Join(updateChannelFlags, ", --"))
		}

		req := client.UpdateStreamRequest{
			StreamID:                   streamID,
			Description:                stringFlag(cmd, "description"),
			NewName:                    stringFlag(cmd, "new-name"),
			IsPrivate:                  boolFlag(cmd, "invite-only"),
			IsWebPublic:                boolFlag(cmd, "is-web-public"),
			HistoryPublicToSubscribers: boolFlag(cmd, "history-public-to-subscribers"),
		}

		if days := stringFlag(cmd, "message-retention-days"); days != nil {
			req.MessageRetentionDays = client.RetentionDays(*days)
		}

		if postingGroup := stringFlag(cmd, "can-send-message-group"); postingGroup != nil {
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
	Use:     "delete-channel [channel]",
	Aliases: []string{"delete-stream"},
	Short:   "Delete a channel",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
		}

		resp, err := zulipClient.DeleteStream(streamID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var listStreamTopicsCmd = &cobra.Command{
	Use:     "list-channel-topics [channel]",
	Aliases: []string{"list-stream-topics"},
	Short:   "List all topics in a channel",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
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
	Long: `Subscribe to one or more channels, creating any that do not exist yet.

With no --principals this subscribes you. Naming other people subscribes them
instead of you, which needs permission to add subscribers to the channel:

  zulip-cli subscribe general --principals 12,dana@example.com

Subscribing someone the caller may not add is an error that abandons the whole
request, unless --authorization-errors-fatal=false, which subscribes everyone
allowed and reports the rest under "unauthorized".`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		description, _ := cmd.Flags().GetString("description")

		var subscriptions []client.ChannelSubscription
		for _, name := range args {
			subscriptions = append(subscriptions, client.ChannelSubscription{
				Name: name, Description: description,
			})
		}

		req := client.SubscribeRequest{
			Subscriptions:            subscriptions,
			AuthorizationErrorsFatal: boolFlag(cmd, "authorization-errors-fatal"),
			Announce:                 boolFlag(cmd, "announce"),
		}

		principals, err := principalsFlag(cmd)
		if err != nil {
			return err
		}
		req.Principals = principals

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
	Long: `Unsubscribe from one or more channels.

With no --principals this unsubscribes you. Naming other people unsubscribes
them instead, which needs permission to remove subscribers from the channel.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		req := client.UnsubscribeRequest{
			Subscriptions: args,
		}

		principals, err := principalsFlag(cmd)
		if err != nil {
			return err
		}
		req.Principals = principals

		resp, err := zulipClient.Unsubscribe(req)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

// principalsFlag reads the people a subscription change is about, given as
// user IDs or email addresses. An empty result means the caller themselves,
// which is what the server does with no principals at all.
func principalsFlag(cmd *cobra.Command) ([]interface{}, error) {
	values, _ := cmd.Flags().GetStringSlice("principals")
	if len(values) == 0 {
		return nil, nil
	}

	userIDs, err := resolveUserIDs(zulipClient, values)
	if err != nil {
		return nil, fmt.Errorf("--principals: %w", err)
	}

	principals := make([]interface{}, 0, len(userIDs))
	for _, userID := range userIDs {
		principals = append(principals, userID)
	}
	return principals, nil
}

var getSubscriptionStatusCmd = &cobra.Command{
	Use:   "get-subscription-status [user] [channel]",
	Short: "Check whether a user is subscribed to a channel",
	Long: `Check whether a user is subscribed to a channel.

The user is a user ID or an email address, and the channel is a channel ID or
a channel name.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		userIDs, err := resolveUserIDs(zulipClient, args[:1])
		if err != nil {
			return err
		}

		streamID, err := resolveChannelID(zulipClient, args[1])
		if err != nil {
			return err
		}

		resp, err := zulipClient.GetSubscriptionStatus(userIDs[0], streamID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var getStreamEmailAddressCmd = &cobra.Command{
	Use:     "get-channel-email-address [channel]",
	Aliases: []string{"get-stream-email-address"},
	Short:   "Get the address that emails messages into a channel",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
		}

		resp, err := zulipClient.GetStreamEmailAddress(streamID)
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

// subscriptionSettingFlags are the per-subscription preferences, each naming
// the flag, the property the server calls it, and how to read the value the
// user typed. These are your own settings for a channel, not the channel's.
var subscriptionSettingFlags = []struct {
	name     string
	property string
	help     string
	value    func(*cobra.Command, string) interface{}
}{
	{"color", "color", "Colour the channel shows in, as a hex code such as #76ce90", stringSettingValue},
	{"pin-to-top", "pin_to_top", "Pin the channel to the top of your list", boolSettingValue},
	{"is-muted", "is_muted", "Mute the channel", boolSettingValue},
	{"desktop-notifications", "desktop_notifications", "Show desktop notifications for the channel", boolSettingValue},
	{"audible-notifications", "audible_notifications", "Play a sound for the channel", boolSettingValue},
	{"push-notifications", "push_notifications", "Send mobile push notifications for the channel", boolSettingValue},
	{"email-notifications", "email_notifications", "Send email notifications for the channel", boolSettingValue},
	{"wildcard-mentions-notify", "wildcard_mentions_notify", "Notify you on @all and @channel in the channel", boolSettingValue},
}

func boolSettingValue(cmd *cobra.Command, name string) interface{} {
	if value := boolFlag(cmd, name); value != nil {
		return *value
	}
	return nil
}

func stringSettingValue(cmd *cobra.Command, name string) interface{} {
	if value := stringFlag(cmd, name); value != nil {
		return *value
	}
	return nil
}

var updateSubscriptionCmd = &cobra.Command{
	Use:     "update-subscription [channel]",
	Aliases: []string{"update-subscription-settings"},
	Short:   "Change your own settings for a channel",
	Long: `Change your own settings for a channel you are subscribed to.

These are personal preferences — the channel's colour, whether it is pinned or
muted, and which notifications it sends — and they change nothing for anyone
else. update-channel is what changes the channel itself.

The channel is a channel ID or a channel name.

  zulip-cli update-subscription general --color '#76ce90' --pin-to-top
  zulip-cli update-subscription general --is-muted=false --push-notifications`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var data []map[string]interface{}
		var names []string

		for _, setting := range subscriptionSettingFlags {
			names = append(names, setting.name)
			value := setting.value(cmd, setting.name)
			if value == nil {
				continue
			}
			data = append(data, map[string]interface{}{
				"property": setting.property,
				"value":    value,
			})
		}

		if len(data) == 0 {
			return fmt.Errorf("nothing to change: pass --%s", strings.Join(names, ", --"))
		}

		// The channel is looked up only once there is something to apply to it.
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
		}
		for _, entry := range data {
			entry["stream_id"] = streamID
		}

		resp, err := zulipClient.UpdateSubscriptionSettings(
			client.UpdateSubscriptionSettingsRequest{SubscriptionData: data})
		if err != nil {
			return err
		}

		return printResult(resp)
	},
}

var addDefaultStreamCmd = &cobra.Command{
	Use:     "add-default-channel [channel]",
	Aliases: []string{"add-default-stream"},
	Short:   "Subscribe new users to a channel automatically",
	Long: `Add a channel to the organization's default channels.

New users are subscribed to it when they join. The channel is a channel ID or
a channel name.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
		}

		resp, err := zulipClient.AddDefaultStream(streamID)
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
	Use:   "list-subscribers [channel]",
	Short: "List subscribers to a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
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
	Use:   "move-topic [channel] [topic]",
	Short: "Move a topic to another channel or rename it",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		streamID, err := resolveChannelID(zulipClient, args[0])
		if err != nil {
			return err
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
	updateStreamCmd.Flags().Bool("invite-only", false, "Make the channel private")
	updateStreamCmd.Flags().Bool("is-web-public", false, "Make the channel readable by anyone on the internet")
	updateStreamCmd.Flags().Bool("history-public-to-subscribers", false,
		"Share history with users who subscribe later")
	updateStreamCmd.Flags().String("message-retention-days", "",
		`Days to keep messages, or "realm_default" or "unlimited"`)
	updateStreamCmd.Flags().String("can-send-message-group", "",
		"Who may post in the channel"+groupFlagHelp)

	subscribeCmd.Flags().String("description", "", "Channel description (for new channels)")
	subscribeCmd.Flags().Bool("announce", false, "Announce any channel this creates")
	subscribeCmd.Flags().Bool("authorization-errors-fatal", true,
		"Fail the whole request if any of --principals may not be subscribed")
	addPrincipalsFlag(subscribeCmd, "subscribe")
	addPrincipalsFlag(unsubscribeCmd, "unsubscribe")

	for _, setting := range subscriptionSettingFlags {
		if setting.property == "color" {
			updateSubscriptionCmd.Flags().String(setting.name, "", setting.help)
			continue
		}
		updateSubscriptionCmd.Flags().Bool(setting.name, false, setting.help)
	}

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

// addPrincipalsFlag registers the people a subscription change is about. The
// default — nobody named — is the caller themselves.
func addPrincipalsFlag(cmd *cobra.Command, verb string) {
	cmd.Flags().StringSlice("principals", nil,
		"User IDs or email addresses to "+verb+" instead of yourself (comma-separated)")
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
